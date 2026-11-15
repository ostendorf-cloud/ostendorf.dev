package webapp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/fstest"
)

// fakeBuild builds the minimal dist and prerendered filesystems
// newPageServer and the handlers it builds expect: one asset, the home
// page's prerendered file, and the site's own 404 page.
func fakeBuild() (dist, prerendered fstest.MapFS) {
	dist = fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
	}
	prerendered = fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<title>home</title>")},
		"404.html":   &fstest.MapFile{Data: []byte("<title>not found</title>")},
	}
	return dist, prerendered
}

func testConfig() Config {
	return Config{SiteURL: &url.URL{Scheme: "https", Host: "ostendorf.dev"}}
}

func TestNewHandlerRejectsADistWithNoAssetsDirectory(t *testing.T) {
	t.Parallel()

	dist := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	prerendered := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<title>home</title>")}}

	if _, err := NewHandler(testConfig(), dist, prerendered); err == nil {
		t.Fatal("NewHandler() error = nil, want an error for a dist with no assets directory")
	}
}

func TestHandlerServesEveryKnownRoute(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "<title>home</title>" {
		t.Errorf("GET / body = %q, want the prerendered home page", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("GET / Content-Type = %q, want text/html; charset=utf-8", ct)
	}
}

func TestHandlerServesContentHashedAssetsAsImmutable(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/app.js", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/app.js = %d, want %d", rec.Code, http.StatusOK)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("GET /assets/app.js Cache-Control = %q, want the immutable directive", cc)
	}
}

func TestHandlerServesTheSitesOwn404PageForAnUnknownPath(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/does-not-exist/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist/ = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.String() != "<title>not found</title>" {
		t.Errorf("GET /does-not-exist/ body = %q, want the prerendered 404 page", rec.Body.String())
	}
}

func TestHandlerFallsBackToAPlainTextLineWhenTheSitesOwn404PageIsMissing(t *testing.T) {
	t.Parallel()

	dist, _ := fakeBuild()
	prerendered := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<title>home</title>")}}
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/does-not-exist/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist/ = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestHandlerRedirectsAKnownRouteMissingItsTrailingSlash exercises the
// redirect against a route table carrying a slug beyond home, because
// the real routes.json currently holds only the home page, whose own
// path, "/", already carries its trailing slash and can never take this
// branch.
func TestHandlerRedirectsAKnownRouteMissingItsTrailingSlash(t *testing.T) {
	t.Parallel()

	_, prerendered := fakeBuild()
	p := &pageServer{prerendered: prerendered, routes: map[string]string{"projects": "projects.html"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/projects?page=2", nil)
	p.pageHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("GET /projects?page=2 = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if loc := rec.Header().Get("Location"); loc != "/projects/?page=2" {
		t.Errorf("GET /projects?page=2 Location = %q, want %q", loc, "/projects/?page=2")
	}
}

// TestHandlerServesTheSitesOwn404PageForAnUnknownPathWithNoTrailingSlash
// guards the trailing slash branch against redirecting a path that is
// not a route at all: /metrics on the public listener is exactly such a
// path, and without the isKnownRoute gate it would get a 301 to a
// target that only then answers 404, instead of going straight there.
func TestHandlerServesTheSitesOwn404PageForAnUnknownPathWithNoTrailingSlash(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /metrics = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.String() != "<title>not found</title>" {
		t.Errorf("GET /metrics body = %q, want the prerendered 404 page", rec.Body.String())
	}
}

func TestHandlerRefusesAMethodOtherThanGetOrHead(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/does-not-exist/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /does-not-exist/ = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("POST /does-not-exist/ Allow = %q, want %q", allow, "GET, HEAD")
	}
}
