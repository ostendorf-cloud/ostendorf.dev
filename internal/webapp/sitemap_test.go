package webapp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRobotsHandlerNamesTheSitemapUnderTheConfiguredSiteURL(t *testing.T) {
	t.Parallel()

	h := &handler{siteURL: &url.URL{Scheme: "https", Host: "ostendorf.dev"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/robots.txt", nil)
	h.robotsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /robots.txt = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Allow: /") {
		t.Errorf("robots.txt = %q, want it to allow every route", body)
	}
	if !strings.Contains(body, "Sitemap: https://ostendorf.dev/sitemap.xml") {
		t.Errorf("robots.txt = %q, want it to name the sitemap under the configured site URL", body)
	}
}

func TestSitemapHandlerListsEveryRouteTheHandlerActuallyServes(t *testing.T) {
	t.Parallel()

	h := &handler{siteURL: &url.URL{Scheme: "https", Host: "ostendorf.dev"}, routes: routeFiles}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sitemap.xml", nil)
	h.sitemapHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /sitemap.xml = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for slug := range routeFiles {
		want := "<loc>" + routeURL("https://ostendorf.dev", slug) + "</loc>"
		if !strings.Contains(body, want) {
			t.Errorf("sitemap.xml = %q, want it to contain %q", body, want)
		}
	}
	if got := strings.Count(body, "<url>"); got != len(routeFiles) {
		t.Errorf("sitemap.xml has %d <url> entries, want exactly %d, one per route", got, len(routeFiles))
	}
}

func TestRouteURL(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		slug string
		want string
	}{
		"home page's empty slug": {slug: "", want: "https://ostendorf.dev/"},
		"a regular slug":         {slug: "projects", want: "https://ostendorf.dev/projects/"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := routeURL("https://ostendorf.dev", tc.slug); got != tc.want {
				t.Errorf("routeURL(%q) = %q, want %q", tc.slug, got, tc.want)
			}
		})
	}
}
