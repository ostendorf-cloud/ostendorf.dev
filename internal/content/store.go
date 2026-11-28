// Package content holds the cv document the data pages render.
//
// It reads the document from cvd on request into an in-memory cache, one
// entry per language, and keeps the last good copy when a read fails. ADR
// 0001 records the policy: no timer, one shared refresh, no wait for a
// request that already has a copy, and a gap after a failed read.
package content

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ostendorf-cloud/cv/client"
)

// The languages the site requests from cvd.
const (
	LangDE = "de"
	LangEN = "en"
)

// DefaultRetryGap is how long the store waits after a failed read before it
// calls cvd again for the same language.
const DefaultRetryGap = 30 * time.Second

// ErrUnavailable is returned when no copy of the document exists yet: cvd is
// not configured, the language is not served, or no read has succeeded.
var ErrUnavailable = errors.New("content: no cv document available")

// Source reads the document of one language once. *client.Client is the
// implementation.
type Source interface {
	Get(ctx context.Context, etag string) (client.Result, error)
}

// Options configures New.
type Options struct {
	// Sources maps a language to the Source that reads it. An empty map is
	// valid: every Get then returns ErrUnavailable.
	Sources map[string]Source
	// TTL is the age after which a request triggers a refresh.
	TTL time.Duration
	// Timeout bounds one read, however long the caller waits. Zero sets no
	// bound beyond the one the Source has.
	Timeout time.Duration
	// RetryGap is the time after a failed read before the next one. Zero
	// means DefaultRetryGap.
	RetryGap time.Duration
	// Now is the clock. Nil means time.Now.
	Now func() time.Time
	// Logger receives one line per failed read, per refused schema version
	// and per recovery. Nil logs nothing.
	Logger *slog.Logger
}

// Store is the cache of the cv document. It is safe for concurrent use.
type Store struct {
	ttl      time.Duration
	timeout  time.Duration
	retryGap time.Duration
	now      func() time.Time
	logger   *slog.Logger

	mu      sync.Mutex
	entries map[string]*entry
	// refreshes counts running reads, so a test can wait for them.
	refreshes sync.WaitGroup
}

// entry is the state of one language. Every field is guarded by Store.mu.
type entry struct {
	source    Source
	doc       *client.Document
	etag      string
	fetchedAt time.Time
	// retryAt is the earliest time of the next read after a failure.
	retryAt time.Time
	// failed is true from a failed read until the next successful one.
	failed bool
	// flight is the read in progress, or nil.
	flight *flight
}

// flight is one read that every waiting request shares.
type flight struct {
	done chan struct{}
}

// New builds a Store.
func New(opts Options) *Store {
	s := &Store{
		ttl:      opts.TTL,
		timeout:  opts.Timeout,
		retryGap: opts.RetryGap,
		now:      opts.Now,
		logger:   opts.Logger,
		entries:  make(map[string]*entry, len(opts.Sources)),
	}
	if s.retryGap <= 0 {
		s.retryGap = DefaultRetryGap
	}
	if s.now == nil {
		s.now = time.Now
	}
	if s.logger == nil {
		s.logger = slog.New(slog.DiscardHandler)
	}
	for lang, src := range opts.Sources {
		s.entries[lang] = &entry{source: src}
	}
	return s
}

// Get returns the document of lang. The caller must not modify it.
//
// A fresh or expired copy is returned at once. An expired copy also starts
// one refresh in the background, unless one runs or the gap after a failed
// read has not passed. Only a call with no copy waits, for the shared read
// and at most until ctx ends. It returns ErrUnavailable when that read fails
// or the gap has not passed.
func (s *Store) Get(ctx context.Context, lang string) (*client.Document, error) {
	s.mu.Lock()
	e, ok := s.entries[lang]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w: language %q is not served", ErrUnavailable, lang)
	}

	now := s.now()
	if e.doc != nil {
		if now.Sub(e.fetchedAt) >= s.ttl {
			s.startLocked(ctx, lang, e, now)
		}
		doc := e.doc
		s.mu.Unlock()
		return doc, nil
	}

	f := s.startLocked(ctx, lang, e, now)
	s.mu.Unlock()
	if f == nil {
		return nil, ErrUnavailable
	}

	select {
	case <-f.done:
	case <-ctx.Done():
		return nil, fmt.Errorf("content: wait for cvd: %w", ctx.Err())
	}

	s.mu.Lock()
	doc := e.doc
	s.mu.Unlock()
	if doc == nil {
		return nil, ErrUnavailable
	}
	return doc, nil
}

// startLocked returns the read in progress for e, or starts one. It returns
// nil when the gap after a failed read has not passed. The caller holds
// s.mu.
func (s *Store) startLocked(ctx context.Context, lang string, e *entry, now time.Time) *flight {
	if e.flight != nil {
		return e.flight
	}
	if now.Before(e.retryAt) {
		return nil
	}

	f := &flight{done: make(chan struct{})}
	e.flight = f
	s.refreshes.Add(1)
	// The read outlives the request that started it: a caller that gives up
	// must not cancel the read the next request will share.
	go s.refresh(context.WithoutCancel(ctx), lang, e, f, e.etag)
	return f
}

// refresh reads the document once and records the outcome.
func (s *Store) refresh(ctx context.Context, lang string, e *entry, f *flight, etag string) {
	defer s.refreshes.Done()

	cancel := func() {}
	if s.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
	}
	res, err := e.source.Get(ctx, etag)
	cancel()

	s.mu.Lock()
	now := s.now()
	e.flight = nil
	if err == nil && res.Document == nil && (!res.NotModified || e.doc == nil) {
		err = errors.New("content: cvd answered with no document")
	}

	var recovered bool
	if err != nil {
		e.retryAt = now.Add(s.retryGap)
		e.failed = true
	} else {
		if !res.NotModified {
			e.doc = res.Document
			e.etag = res.ETag
		}
		e.fetchedAt = now
		e.retryAt = time.Time{}
		recovered = e.failed
		e.failed = false
	}
	copyHeld := e.doc != nil
	s.mu.Unlock()
	close(f.done)

	switch {
	case err != nil:
		s.logFailure(lang, err, copyHeld)
	case recovered:
		s.logger.Info("cvd read recovered", "lang", lang)
	}
}

// logFailure writes the one line a failed read gets. It never carries the
// address of cvd: the client's errors hold none.
func (s *Store) logFailure(lang string, err error, copyHeld bool) {
	var version *client.SchemaVersionError
	if errors.As(err, &version) {
		s.logger.Error("cvd document refused: unknown schema version",
			"lang", lang, "got", version.Got, "supported", version.Supported, "copy_held", copyHeld)
		return
	}
	s.logger.Warn("cvd read failed", "lang", lang, "err", err, "copy_held", copyHeld)
}
