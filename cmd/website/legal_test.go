package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

// TestPrerenderedSitemapIsTheServedSitemap proves the sitemap govue-check-legal
// reads, written by the prerender, is the document the server answers for the
// default origin. Without it the check could pass on a sitemap nobody serves.
func TestPrerenderedSitemapIsTheServedSitemap(t *testing.T) {
	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub() error = %v", err)
	}
	built, err := fs.ReadFile(sub, "sitemap.xml")
	if err != nil {
		t.Fatalf("prerendered has no sitemap.xml: %v", err)
	}

	site, err := url.Parse(config.DefaultSiteURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	cfg := testCfg()
	cfg.SiteURL = site
	h, err := buildHandler(cfg, slog.New(slog.DiscardHandler), nil, nil)
	if err != nil {
		t.Fatalf("buildHandler() error = %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sitemap.xml", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /sitemap.xml = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != string(built) {
		t.Fatalf("served sitemap differs from the prerendered one\nserved:\n%s\nprerendered:\n%s", got, built)
	}
}

// TestLegalPagesStateTheOperatorAndNoDisputeResolution reads the shipped
// bytes of the imprint in both languages.
func TestLegalPagesStateTheOperatorAndNoDisputeResolution(t *testing.T) {
	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub() error = %v", err)
	}
	for _, lang := range []string{"de", "en"} {
		data, err := fs.ReadFile(sub, lang+"/imprint/index.html")
		if err != nil {
			t.Fatalf("no %s imprint: %v", lang, err)
		}
		page := string(data)
		for _, fact := range []string{"Simon Ostendorf", "Frankenstr. 8", "52070 Aachen", "+49 1573 0971784", "simon@ostendorf.dev", "DE362164357"} {
			if !strings.Contains(page, fact) {
				t.Errorf("%s imprint lacks %q", lang, fact)
			}
		}
		for _, banned := range []string{"Verbraucherstreit", "Streitbeilegung", "dispute resolution", "arbitration"} {
			if strings.Contains(page, banned) {
				t.Errorf("%s imprint contains %q", lang, banned)
			}
		}
	}
}
