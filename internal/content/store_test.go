package content

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/cv/client"
)

const (
	testTTL     = 5 * time.Minute
	testTimeout = 5 * time.Second
)

// fakeClock is a clock a test moves by hand.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// reply is one canned answer of fakeSource.
type reply struct {
	res client.Result
	err error
}

// fakeSource answers from a queue of replies and records the ETag of every
// call. When block is set, a call waits for it before it answers.
type fakeSource struct {
	mu      sync.Mutex
	replies []reply
	etags   []string
	block   chan struct{}
	// started receives one value per call that reached the source.
	started chan struct{}
}

func newFakeSource(replies ...reply) *fakeSource {
	return &fakeSource{replies: replies, started: make(chan struct{}, 16)}
}

func (f *fakeSource) Get(ctx context.Context, etag string) (client.Result, error) {
	f.mu.Lock()
	f.etags = append(f.etags, etag)
	var r reply
	if len(f.replies) > 0 {
		r = f.replies[0]
		f.replies = f.replies[1:]
	}
	block := f.block
	f.mu.Unlock()

	f.started <- struct{}{}
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return client.Result{}, ctx.Err()
		}
	}
	return r.res, r.err
}

func (f *fakeSource) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.etags)
}

func (f *fakeSource) sentETags() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.etags...)
}

func doc(repo string) *client.Document {
	return &client.Document{
		SchemaVersion: 1,
		Language:      LangEN,
		Projects:      []client.Project{{ID: repo, GithubRepo: "ostendorf-cloud/" + repo}},
	}
}

func okReply(d *client.Document, etag string) reply {
	return reply{res: client.Result{Document: d, ETag: etag}}
}

// record is one log line a test handler captured.
type record struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

// recorder is a slog handler that keeps what it is given.
type recorder struct {
	mu      sync.Mutex
	records []record
}

func (r *recorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *recorder) Handle(_ context.Context, rec slog.Record) error {
	attrs := map[string]any{}
	rec.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record{level: rec.Level, msg: rec.Message, attrs: attrs})
	return nil
}

func (r *recorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *recorder) WithGroup(string) slog.Handler      { return r }

func (r *recorder) all() []record {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]record(nil), r.records...)
}

func newTestStore(clock *fakeClock, log *recorder, sources map[string]Source) *Store {
	return New(Options{
		Sources: sources,
		TTL:     testTTL,
		Timeout: testTimeout,
		Now:     clock.Now,
		Logger:  slog.New(log),
	})
}

func TestStore_Get_withoutASource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sources map[string]Source
		lang    string
	}{
		{name: "cvd is not configured", sources: nil, lang: LangEN},
		{name: "the language is not served", sources: map[string]Source{LangEN: newFakeSource()}, lang: "fr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newTestStore(newFakeClock(), &recorder{}, tc.sources)
			_, err := s.Get(t.Context(), tc.lang)
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("Get() error = %v, want ErrUnavailable", err)
			}
		})
	}
}

func TestStore_Get_firstReadWaitsForTheDocument(t *testing.T) {
	t.Parallel()

	want := doc("cv")
	src := newFakeSource(okReply(want, `"v1"`))
	s := newTestStore(newFakeClock(), &recorder{}, map[string]Source{LangEN: src})

	got, err := s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != want {
		t.Errorf("Get() = %p, want the document the source returned", got)
	}
}

func TestStore_Get_servesTheCopyUntilTheLifetimeEnds(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	src := newFakeSource(okReply(doc("first"), `"v1"`), okReply(doc("second"), `"v2"`))
	s := newTestStore(clock, &recorder{}, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}

	clock.Advance(testTTL - time.Second)
	got, err := s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() before expiry error = %v", err)
	}
	s.refreshes.Wait()
	if n := src.calls(); n != 1 {
		t.Fatalf("source calls before expiry = %d, want 1", n)
	}
	if got.Projects[0].ID != "first" {
		t.Errorf("document before expiry = %q, want first", got.Projects[0].ID)
	}
}

func TestStore_Get_firstRequestAfterExpiryRefreshesWithoutWaiting(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	src := newFakeSource(okReply(doc("first"), `"v1"`), okReply(doc("second"), `"v2"`))
	s := newTestStore(clock, &recorder{}, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}
	<-src.started

	src.mu.Lock()
	src.block = make(chan struct{})
	release := src.block
	src.mu.Unlock()

	clock.Advance(testTTL)
	got, err := s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() after expiry error = %v", err)
	}
	if got.Projects[0].ID != "first" {
		t.Errorf("document served during the refresh = %q, want the expired copy", got.Projects[0].ID)
	}
	<-src.started

	close(release)
	s.refreshes.Wait()

	got, err = s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() after refresh error = %v", err)
	}
	if got.Projects[0].ID != "second" {
		t.Errorf("document after the refresh = %q, want second", got.Projects[0].ID)
	}
	if n := src.calls(); n != 2 {
		t.Errorf("source calls = %d, want 2", n)
	}
}

func TestStore_Get_sendsTheETagAndKeepsTheCopyOnNotModified(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	first := doc("first")
	src := newFakeSource(
		okReply(first, `"v1"`),
		reply{res: client.Result{ETag: `"v1"`, NotModified: true}},
	)
	s := newTestStore(clock, &recorder{}, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}
	clock.Advance(testTTL)
	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("Get() after expiry error = %v", err)
	}
	s.refreshes.Wait()

	// The 304 renewed the lifetime: no third read inside it.
	clock.Advance(testTTL - time.Second)
	got, err := s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() after the 304 error = %v", err)
	}
	s.refreshes.Wait()

	if got != first {
		t.Errorf("document after a 304 = %p, want the held copy", got)
	}
	etags := src.sentETags()
	if len(etags) != 2 || etags[0] != "" || etags[1] != `"v1"` {
		t.Errorf("ETags sent = %q, want [\"\" \"\\\"v1\\\"\"]", etags)
	}
}

func TestStore_Get_concurrentFirstRequestsShareOneRead(t *testing.T) {
	t.Parallel()

	src := newFakeSource(okReply(doc("cv"), `"v1"`))
	src.block = make(chan struct{})
	s := newTestStore(newFakeClock(), &recorder{}, map[string]Source{LangEN: src})

	const callers = 8
	errs := make(chan error, callers)
	for range callers {
		go func() {
			_, err := s.Get(t.Context(), LangEN)
			errs <- err
		}()
	}
	<-src.started
	close(src.block)

	for range callers {
		if err := <-errs; err != nil {
			t.Errorf("Get() error = %v", err)
		}
	}
	if n := src.calls(); n != 1 {
		t.Errorf("source calls = %d, want 1", n)
	}
}

func TestStore_Get_aFailedFirstReadWaitsForTheGap(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	src := newFakeSource(
		reply{err: &client.StatusError{StatusCode: 503}},
		okReply(doc("cv"), `"v1"`),
	)
	s := newTestStore(clock, &recorder{}, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() after a 5xx error = %v, want ErrUnavailable", err)
	}

	clock.Advance(DefaultRetryGap - time.Second)
	if _, err := s.Get(t.Context(), LangEN); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() inside the gap error = %v, want ErrUnavailable", err)
	}
	if n := src.calls(); n != 1 {
		t.Fatalf("source calls inside the gap = %d, want 1", n)
	}

	clock.Advance(time.Second)
	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("Get() after the gap error = %v, want the document", err)
	}
	if n := src.calls(); n != 2 {
		t.Errorf("source calls after the gap = %d, want 2", n)
	}
}

func TestStore_Get_keepsTheLastGoodCopyWhenARefreshFails(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	first := doc("first")
	src := newFakeSource(
		okReply(first, `"v1"`),
		reply{err: errors.New("connection refused")},
		okReply(doc("second"), `"v2"`),
	)
	log := &recorder{}
	s := newTestStore(clock, log, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}

	// The refresh fails: the copy stays, far beyond the lifetime.
	clock.Advance(testTTL)
	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("Get() during the failing refresh error = %v", err)
	}
	s.refreshes.Wait()

	clock.Advance(DefaultRetryGap - time.Second)
	got, err := s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() inside the gap error = %v", err)
	}
	s.refreshes.Wait()
	if got != first {
		t.Errorf("document after a failed refresh = %p, want the last good copy", got)
	}
	if n := src.calls(); n != 2 {
		t.Errorf("source calls inside the gap = %d, want 2", n)
	}

	// After the gap the next request retries and the site recovers.
	clock.Advance(time.Second)
	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("Get() after the gap error = %v", err)
	}
	s.refreshes.Wait()
	got, err = s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() after recovery error = %v", err)
	}
	if got.Projects[0].ID != "second" {
		t.Errorf("document after recovery = %q, want second", got.Projects[0].ID)
	}

	records := log.all()
	if len(records) != 2 {
		t.Fatalf("log lines = %d, want one for the failure and one for the recovery", len(records))
	}
	if records[0].level != slog.LevelWarn || records[0].attrs["copy_held"] != true {
		t.Errorf("failure line = %+v, want a warning that says a copy is held", records[0])
	}
	if records[1].level != slog.LevelInfo || records[1].attrs["lang"] != LangEN {
		t.Errorf("recovery line = %+v, want an info line for en", records[1])
	}
}

func TestStore_Get_refusesAnUnknownSchemaVersionAndKeepsTheCopy(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	first := doc("first")
	src := newFakeSource(
		okReply(first, `"v1"`),
		reply{err: &client.SchemaVersionError{Got: 2, Supported: 1}},
	)
	log := &recorder{}
	s := newTestStore(clock, log, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}
	clock.Advance(testTTL)
	got, err := s.Get(t.Context(), LangEN)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	s.refreshes.Wait()

	if got != first {
		t.Errorf("document = %p, want the last good copy", got)
	}
	records := log.all()
	if len(records) != 1 {
		t.Fatalf("log lines = %d, want 1", len(records))
	}
	if records[0].level != slog.LevelError || records[0].attrs["got"] != int64(2) || records[0].attrs["supported"] != int64(1) {
		t.Errorf("log line = %+v, want an error line naming both versions", records[0])
	}
}

func TestStore_Get_aSlowReadEndsAtTheTimeout(t *testing.T) {
	t.Parallel()

	src := newFakeSource(okReply(doc("cv"), `"v1"`))
	src.block = make(chan struct{})
	s := New(Options{
		Sources: map[string]Source{LangEN: src},
		TTL:     testTTL,
		Timeout: 20 * time.Millisecond,
		Logger:  slog.New(&recorder{}),
	})

	if _, err := s.Get(t.Context(), LangEN); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() error = %v, want ErrUnavailable once the read timed out", err)
	}
}

func TestStore_Get_aWaitingCallerThatGivesUpDoesNotCancelTheRead(t *testing.T) {
	t.Parallel()

	src := newFakeSource(okReply(doc("cv"), `"v1"`))
	src.block = make(chan struct{})
	s := newTestStore(newFakeClock(), &recorder{}, map[string]Source{LangEN: src})

	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		_, err := s.Get(ctx, LangEN)
		errs <- err
	}()
	<-src.started
	cancel()
	if err := <-errs; !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() error = %v, want context.Canceled", err)
	}

	close(src.block)
	s.refreshes.Wait()

	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("Get() after the read finished error = %v, want the document", err)
	}
	if n := src.calls(); n != 1 {
		t.Errorf("source calls = %d, want 1", n)
	}
}

func TestStore_Get_keepsEachLanguageApart(t *testing.T) {
	t.Parallel()

	de := newFakeSource(reply{err: &client.StatusError{StatusCode: 500}})
	en := newFakeSource(okReply(doc("cv"), `"v1"`))
	s := newTestStore(newFakeClock(), &recorder{}, map[string]Source{LangDE: de, LangEN: en})

	if _, err := s.Get(t.Context(), LangDE); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get(de) error = %v, want ErrUnavailable", err)
	}
	if _, err := s.Get(t.Context(), LangEN); err != nil {
		t.Fatalf("Get(en) error = %v, want the document: a failing de must not gate en", err)
	}
}

func TestStore_Get_answersWithNoDocumentAsAFailedRead(t *testing.T) {
	t.Parallel()

	src := newFakeSource(reply{res: client.Result{}})
	s := newTestStore(newFakeClock(), &recorder{}, map[string]Source{LangEN: src})

	if _, err := s.Get(t.Context(), LangEN); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() error = %v, want ErrUnavailable", err)
	}
}
