package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ostendorf-cloud/govue/health"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

func testCfg() config.Config {
	return config.Config{SiteURL: &url.URL{Scheme: "https", Host: "ostendorf.dev"}}
}

// TestEmbeddedDistBuildsAWorkingHandler proves the embedded dist and
// prerendered filesystems are shaped the way buildHandler expects, without
// starting a server or binding a port: a build with an empty or mis-shaped
// frontend build fails here instead of at a request.
func TestEmbeddedDistBuildsAWorkingHandler(t *testing.T) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(distFS, \"dist\") error = %v", err)
	}

	if _, err := buildHandler(testCfg()); err != nil {
		t.Fatalf("buildHandler() error = %v", err)
	}

	if _, err := fs.Stat(dist, "index.html"); err != nil {
		t.Fatalf("embedded dist has no index.html: %v", err)
	}
}

// TestBuildAdminHandlerServesHealthReadinessAndMetrics proves the admin
// handler answers every route the chart's probes and a Prometheus scraper
// depend on, without binding a port.
func TestBuildAdminHandlerServesHealthReadinessAndMetrics(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want int
	}{
		{name: "liveness", path: "/healthz", want: http.StatusOK},
		{name: "readiness with no checks registered", path: "/readyz", want: http.StatusOK},
		{name: "prometheus scrape", path: "/metrics", want: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := buildAdminHandler(health.New())

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("GET %s = %d, want %d", tc.path, rec.Code, tc.want)
			}
		})
	}
}
