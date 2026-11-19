package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
)

const (
	testWebsiteID = "0a1b2c3d-0000-4000-8000-000000000001"
	testUmamiURL  = "https://umami.example.test"
	visitorUA     = "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"
	// visitorAddr is a documentation address that counts as public.
	visitorAddr = "203.0.113.7:4000"
)

// fakeUmami is the transport standing in for Umami: it records every body
// posted to it and opens no socket. When block is set, a post waits for it
// or for the request context.
type fakeUmami struct {
	mu    sync.Mutex
	posts []string
	block chan struct{}
}

func (f *fakeUmami) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.block != nil {
		select {
		case <-f.block:
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	body, _ := io.ReadAll(req.Body)
	f.mu.Lock()
	f.posts = append(f.posts, string(body))
	f.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func (f *fakeUmami) bodies() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.posts...)
}

func newViews(t *testing.T, fake *fakeUmami) *analytics.Client {
	t.Helper()
	views, err := analytics.New(
		analytics.Config{URL: testUmamiURL, WebsiteID: testWebsiteID},
		analytics.Options{
			Hostname:   "ostendorf.dev",
			Logger:     slog.New(slog.DiscardHandler),
			HTTPClient: &http.Client{Transport: fake},
		})
	if err != nil {
		t.Fatalf("analytics.New() error = %v", err)
	}
	return views
}

// settle sends what is queued, so fake.bodies() is final.
func settle(t *testing.T, views *analytics.Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := views.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func visit(t *testing.T, h http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	req.RemoteAddr = visitorAddr
	req.Header.Set("User-Agent", visitorUA)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func viewURLs(t *testing.T, bodies []string) []string {
	t.Helper()
	var urls []string
	for _, b := range bodies {
		var ev struct {
			Payload struct {
				URL string `json:"url"`
			} `json:"payload"`
		}
		if err := json.Unmarshal([]byte(b), &ev); err != nil {
			t.Fatalf("Umami body %q is not JSON: %v", b, err)
		}
		urls = append(urls, ev.Payload.URL)
	}
	return urls
}

func TestHandlerCountsOnlyPagesAndReportsThePatternNotThePath(t *testing.T) {
	t.Parallel()

	fake := &fakeUmami{}
	views := newViews(t, fake)
	dist, prerendered := fakeBuild()
	cfg := testConfig()
	cfg.Views = views
	h, err := NewHandler(cfg, dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	// Counted: the home page, once, with a query string and a token in it.
	visit(t, h, http.MethodGet, "/?token=s3cret&utm_source=x", nil)
	// Not counted.
	visit(t, h, http.MethodHead, "/", nil)                                     // HEAD
	visit(t, h, http.MethodGet, "/", http.Header{"Dnt": {"1"}})                // Do Not Track
	visit(t, h, http.MethodGet, "/", http.Header{"Sec-Gpc": {"1"}})            // Global Privacy Control
	visit(t, h, http.MethodGet, "/", http.Header{"Sec-Purpose": {"prefetch"}}) // prefetch
	visit(t, h, http.MethodGet, "/assets/app.js", nil)                         // static asset
	visit(t, h, http.MethodGet, "/robots.txt", nil)                            // not HTML
	visit(t, h, http.MethodGet, "/sitemap.xml", nil)                           // not HTML
	visit(t, h, http.MethodGet, "/nowhere/?token=s3cret", nil)                 // 404 page
	visit(t, h, http.MethodGet, "/nowhere", nil)                               // 404 page
	visit(t, h, http.MethodPost, "/", nil)                                     // 405

	settle(t, views)
	bodies := fake.bodies()

	got := viewURLs(t, bodies)
	if len(got) != 1 || got[0] != "/" {
		t.Fatalf("reported URLs = %v, want exactly [/]", got)
	}
	for _, b := range bodies {
		for _, secret := range []string{"s3cret", "token", "utm_source", "nowhere", "?"} {
			if strings.Contains(b, secret) {
				t.Errorf("Umami body %s carries %q", b, secret)
			}
		}
	}
}

func TestHandlerCountsNothingWithoutAClient(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	for name, views := range map[string]*analytics.Client{"nil": nil, "inert": inertViews(t)} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := testConfig()
			cfg.Views = views
			h, err := NewHandler(cfg, dist, prerendered)
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}
			if rec := visit(t, h, http.MethodGet, "/", nil); rec.Code != http.StatusOK {
				t.Fatalf("GET / = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func inertViews(t *testing.T) *analytics.Client {
	t.Helper()
	views, err := analytics.New(analytics.Config{}, analytics.Options{})
	if err != nil {
		t.Fatalf("analytics.New() error = %v", err)
	}
	if views.Enabled() {
		t.Fatal("a client with no configuration is enabled")
	}
	return views
}

// TestRequestWithNoRoutePatternCountsNothing puts a middleware that
// replaces the request between the page view middleware and the mux, as a
// request id or security header middleware would. The mux then sets its
// pattern on a copy the outer middleware never sees, and nothing may be
// reported: the raw path is never a fallback.
func TestRequestWithNoRoutePatternCountsNothing(t *testing.T) {
	t.Parallel()

	fake := &fakeUmami{}
	views := newViews(t, fake)
	dist, prerendered := fakeBuild()
	mux, err := newMux(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("newMux() error = %v", err)
	}
	type marker struct{}
	replacing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), marker{}, true)))
	})
	h := views.Middleware(replacing)

	if rec := visit(t, h, http.MethodGet, "/?token=s3cret", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", rec.Code, http.StatusOK)
	}
	settle(t, views)
	if got := fake.bodies(); len(got) != 0 {
		t.Fatalf("Umami received %v for a request with no route pattern, want nothing", got)
	}
}

// TestSlowUmamiNeverDelaysAResponse blocks every post to Umami and checks
// that responses still come back at once and unchanged.
func TestSlowUmamiNeverDelaysAResponse(t *testing.T) {
	t.Parallel()

	fake := &fakeUmami{block: make(chan struct{})}
	views := newViews(t, fake)
	dist, prerendered := fakeBuild()
	cfg := testConfig()
	cfg.Views = views
	h, err := NewHandler(cfg, dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// More requests than the default queue holds, so a full queue is
		// exercised too.
		for range analytics.DefaultQueueSize + 50 {
			if rec := visit(t, h, http.MethodGet, "/", nil); rec.Code != http.StatusOK || rec.Body.String() != "<title>home</title>" {
				t.Errorf("GET / = %d %q, want the page unchanged", rec.Code, rec.Body.String())
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("responses were held back by a blocked Umami")
	}

	// Shutdown is bounded as well: the posts in flight are cancelled.
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if err := views.Close(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close() error = %v", err)
	}
}
