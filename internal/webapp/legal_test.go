package webapp

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

// legalBuild is fakeBuild plus the four legal pages the route table lists.
func legalBuild() (dist, prerendered fstest.MapFS) {
	dist, prerendered = fakeBuild()
	for _, page := range []string{"de/imprint", "en/imprint", "de/privacy", "en/privacy"} {
		prerendered[page+"/index.html"] = &fstest.MapFile{Data: []byte("<title>" + page + "</title>")}
	}
	return dist, prerendered
}

func TestLegalPathsRedirectToTheLocalePage(t *testing.T) {
	t.Parallel()

	dist, prerendered := legalBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	tests := []struct {
		name   string
		path   string
		header http.Header
		want   string
	}{
		{"imprint without a preference", "/imprint", nil, "/de/imprint"},
		{"imprint for an English visitor", "/imprint", http.Header{"Accept-Language": {"en"}}, "/en/imprint"},
		{"privacy for an English visitor", "/privacy", http.Header{"Accept-Language": {"en"}}, "/en/privacy"},
		{"impressum is German", "/impressum", http.Header{"Accept-Language": {"en"}}, "/de/imprint"},
		{"datenschutz is German", "/datenschutz", http.Header{"Accept-Language": {"en"}}, "/de/privacy"},
		{"datenschutzerklaerung is German", "/datenschutzerklaerung", http.Header{"Accept-Language": {"en"}}, "/de/privacy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := visit(t, h, http.MethodGet, tt.path, tt.header)
			if rec.Code != http.StatusFound {
				t.Fatalf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusFound)
			}
			if got := rec.Header().Get("Location"); got != tt.want {
				t.Fatalf("GET %s Location = %q, want %q", tt.path, got, tt.want)
			}
			// The redirect target must answer 200 in the same handler.
			if target := visit(t, h, http.MethodGet, tt.want, nil); target.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d", tt.want, target.Code, http.StatusOK)
			}
		})
	}
}

// The site has no terms, so neither path may exist: a redirect to a page
// the build does not write would be a dead link.
func TestTermsPathsAreNotServed(t *testing.T) {
	t.Parallel()

	dist, prerendered := legalBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	for _, path := range []string{"/terms", "/agb", "/de/terms", "/en/terms"} {
		rec := visit(t, h, http.MethodGet, path, nil)
		if rec.Code != http.StatusNotFound || rec.Header().Get("Location") != "" {
			t.Errorf("GET %s = %d with Location %q, want a plain 404", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}

// govue's footer links and build check use the path without a slash, and the
// server answers a trailing slash with a 404, not a redirect.
func TestLegalPagesAreServedAtTheExactPath(t *testing.T) {
	t.Parallel()

	dist, prerendered := legalBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	rec := visit(t, h, http.MethodGet, "/en/privacy", nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "<title>en/privacy</title>" {
		t.Fatalf("GET /en/privacy = %d %q, want the prerendered page", rec.Code, rec.Body.String())
	}
	for _, path := range []string{"/en/privacy/", "/en/privacy/index.html", "/de/impressum"} {
		if rec := visit(t, h, http.MethodGet, path, nil); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

func TestSitemapListsTheLegalPagesWithoutATrailingSlash(t *testing.T) {
	t.Parallel()

	dist, prerendered := legalBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	body := visit(t, h, http.MethodGet, "/sitemap.xml", nil).Body.String()
	for _, page := range []string{"de/imprint", "en/imprint", "de/privacy", "en/privacy"} {
		if want := "<loc>https://ostendorf.dev/" + page + "</loc>"; !strings.Contains(body, want) {
			t.Errorf("sitemap.xml lacks %s", want)
		}
	}
}

// The aliases only redirect, and a redirect is not a page. The pages the
// visitor lands on are counted under their own pattern.
func TestLegalRedirectsAreNotCountedButThePagesAre(t *testing.T) {
	t.Parallel()

	fake := &fakeUmami{}
	views := newViews(t, fake)
	dist, prerendered := legalBuild()
	cfg := testConfig()
	cfg.Views = views
	h, err := NewHandler(cfg, dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	for _, path := range []string{"/impressum", "/datenschutz", "/datenschutzerklaerung", "/imprint", "/privacy", "/terms", "/agb"} {
		visit(t, h, http.MethodGet, path, nil)
	}
	settle(t, views)
	if got := viewURLs(t, fake.bodies()); len(got) != 0 {
		t.Fatalf("redirects reported %v, want nothing", got)
	}

	fake2 := &fakeUmami{}
	views2 := newViews(t, fake2)
	cfg.Views = views2
	h, err = NewHandler(cfg, dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	visit(t, h, http.MethodGet, "/de/imprint", nil)
	settle(t, views2)
	got := viewURLs(t, fake2.bodies())
	if len(got) != 1 || got[0] != "/de/imprint" {
		t.Fatalf("reported URLs = %v, want exactly [/de/imprint]", got)
	}
}
