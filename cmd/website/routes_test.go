package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/routes"
)

// TestPrerenderedOutputMatchesRouteTable is the check that proves
// internal/webapp's page handler, sitemap and mux registration, and
// web/scripts/prerender.ts's static renderer, cannot drift apart any
// more. Both now read the same embedded route table (see
// internal/routes), but reading the same file does not by itself prove
// prerender.ts still writes what that table says: this test checks the
// real, generated prerendered output "task prerender" wrote against that
// same table, catching exactly the two failure modes the two-list
// version of this route table used to allow silently:
//
//   - a route the table lists with no prerendered file for it (the "Go
//     only" case: a route would register and the sitemap would
//     advertise it, but a crawler following that link would hit a
//     404);
//   - a prerendered file that answers no route the table lists (the
//     "TypeScript only" case: a page prerender.ts would write under its
//     own pattern, with nothing in internal/webapp ever serving it
//     there).
func TestPrerenderedOutputMatchesRouteTable(t *testing.T) {
	table, err := routes.Load()
	if err != nil {
		t.Fatalf("routes.Load() error = %v", err)
	}
	wantFiles := make(map[string]bool, len(table))
	for _, r := range table {
		wantFiles[r.File] = true
	}

	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}

	for _, r := range table {
		if _, err := fs.Stat(sub, r.File); err != nil {
			t.Errorf("route %q (slug %q) has no prerendered file at %q: %v", r.Name, r.Slug, r.File, err)
		}
	}

	for file := range foundFiles(t, sub) {
		if file == "404.html" {
			// Not a route: the site's own fallback page, served
			// directly by serveNotFound rather than through the
			// route table.
			continue
		}
		if file == "unavailable.html" {
			// Not a route: the 503 placeholder a data page answers with
			// while cvd has delivered no document, served by
			// serveUnavailable rather than through the route table.
			continue
		}
		if file == "sitemap.xml" {
			// Not a page: the build time copy of the sitemap that
			// govue-check-legal reads. TestPrerenderedSitemapIsTheServedSitemap
			// proves it equals what the server answers.
			continue
		}
		if !wantFiles[file] {
			t.Errorf("prerendered file %q answers no route in routes.json", file)
		}
	}
}

// foundFiles lists every file actually prerendered under sub.
func foundFiles(t *testing.T, sub fs.FS) map[string]bool {
	t.Helper()
	found := make(map[string]bool)
	err := fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		found[p] = true
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir(prerendered) error = %v", err)
	}
	if len(found) == 0 {
		t.Fatal("prerendered has no files; run task prerender first")
	}
	return found
}

// TestPrerenderedPagesPinTheTheme guards the website's first paint. A
// visitor with JavaScript disabled, or reading the page before the
// bundle has hydrated, sees whatever data-theme the server-rendered
// markup carries; without it, govue's stylesheets have no palette to
// select and fall back to base.css's generic tokens. This test reads the
// real output "task prerender" wrote, so the attribute has to survive in
// the shipped bytes rather than only in the script that writes it.
func TestPrerenderedPagesPinTheTheme(t *testing.T) {
	const wantTheme = `data-theme="ostendorf-dev"`

	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}

	pages := 0
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		pages++

		data, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		tag := openingHTMLTag(string(data))
		if tag == "" {
			t.Errorf("%s has no opening <html> tag", p)
			return nil
		}
		if !strings.Contains(tag, wantTheme) {
			t.Errorf("%s: <html> tag %q does not carry %s", p, tag, wantTheme)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir(prerendered) error = %v", err)
	}
	if pages == 0 {
		t.Fatal("prerendered has no pages; run task prerender first")
	}
}

// openingHTMLTag is the document's <html ...> tag, attributes and all, or
// an empty string when there is none. It cuts on the raw markup rather
// than parsing the document, because the attributes are the thing under
// test and a parser would be free to normalise them.
func openingHTMLTag(document string) string {
	start := strings.Index(document, "<html")
	if start == -1 {
		return ""
	}
	end := strings.Index(document[start:], ">")
	if end == -1 {
		return ""
	}
	return document[start : start+end+1]
}
