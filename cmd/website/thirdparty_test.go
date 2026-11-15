package main

import (
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestBuiltOutputMakesNoThirdPartyRequest guards the property this site
// inherited from old_ostendorf.dev, a plain HTML page with no JavaScript
// and no third party request: see AGENTS.md. It reads the real,
// generated dist/ and prerendered/ output "task web:build" and
// "task prerender" wrote, the same bytes a browser is served, rather
// than the source that produced them, so a dependency that bakes a
// third party host into the build (a web font, an icon, an analytics
// script) fails here instead of only being noticed in production.
//
// It checks every element whose src or href the browser fetches on its
// own the moment the page loads, with no click required: script, link,
// img, source, iframe, video, audio, embed, object and picture. A plain
// anchor (<a href>) is deliberately excluded: the home page links to
// GitHub, LinkedIn and a mailto: address on purpose (see AGENTS.md), and
// a visitor clicking one of those is their own choice to leave, not a
// request this site's own page causes by itself. A <link> is checked
// only when its own rel is one the browser actually fetches
// (stylesheet, icon, preload, …): rel="canonical" and rel="alternate"
// name another URL without ever fetching it.
//
// srcset on img, source and picture is checked candidate by candidate:
// it is a comma-separated list of "<url> <descriptor>" pairs, not one
// URL, so each candidate's own URL is pulled out and checked on its own
// rather than matching the attribute's raw text.
//
// It also reads every built CSS file, and every inline <style> block
// inside an HTML document, for an absolute @import or url() pointing
// off this site's own origin, since a stylesheet can pull in a web font
// or a background image from a third party as easily as a <link> can,
// whether that stylesheet lives in its own file or inline in the page.
func TestBuiltOutputMakesNoThirdPartyRequest(t *testing.T) {
	t.Parallel()

	checkHTMLTree(t, distFS, "dist")
	checkHTMLTree(t, prerenderedFS, "prerendered")
	checkCSSTree(t, distFS, "dist")
}

// fetchingLinkRels are the only <link> rel values a browser fetches on
// its own; every other rel (canonical, alternate, …) only names a URL.
var fetchingLinkRels = map[string]bool{
	"stylesheet":       true,
	"icon":             true,
	"shortcut icon":    true,
	"apple-touch-icon": true,
	"mask-icon":        true,
	"preload":          true,
	"prefetch":         true,
	"dns-prefetch":     true,
	"preconnect":       true,
	"modulepreload":    true,
	"manifest":         true,
}

var (
	resourceTag = regexp.MustCompile(`(?i)<(script|link|img|source|iframe|video|audio|embed|object|picture)\b[^>]*>`)
	tagAttr     = regexp.MustCompile(`(?i)\b(src|href|rel|srcset)\s*=\s*"([^"]*)"`)
	cssExternal = regexp.MustCompile(`(?i)url\(\s*["']?(https?://[^"')]+)["']?\)|@import\s+["']?(https?://[^"');]+)["']?`)
	inlineStyle = regexp.MustCompile(`(?is)<style\b[^>]*>(.*?)</style>`)
)

func checkHTMLTree(t *testing.T, tree fs.FS, root string) {
	t.Helper()

	found := false
	err := fs.WalkDir(tree, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		found = true
		data, err := fs.ReadFile(tree, p)
		if err != nil {
			return err
		}
		checkHTML(t, p, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir(%q) error = %v", root, err)
	}
	if !found {
		t.Fatalf("%q has no .html file; run task web:build and task prerender first", root)
	}
}

func checkHTML(t *testing.T, path, document string) {
	t.Helper()

	for _, tag := range resourceTag.FindAllString(document, -1) {
		attrs := map[string]string{}
		for _, m := range tagAttr.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(m[1])] = m[2]
		}

		isLink := strings.HasPrefix(strings.ToLower(tag), "<link")
		if isLink && !fetchingLinkRels[strings.ToLower(attrs["rel"])] {
			continue
		}

		for _, key := range []string{"src", "href"} {
			if url, ok := attrs[key]; ok && isThirdParty(url) {
				t.Errorf("%s: %q loads %q from a third party", path, tag, url)
			}
		}

		if srcset, ok := attrs["srcset"]; ok {
			for _, url := range srcsetCandidateURLs(srcset) {
				if isThirdParty(url) {
					t.Errorf("%s: %q loads %q from a third party via srcset", path, tag, url)
				}
			}
		}
	}

	for _, m := range inlineStyle.FindAllStringSubmatch(document, -1) {
		checkCSS(t, path, m[1])
	}
}

// srcsetCandidateURLs splits a srcset attribute into its candidate
// URLs, dropping each candidate's own width or pixel density
// descriptor. srcset is a comma-separated list of image candidate
// strings, each "<url>" optionally followed by whitespace and a
// descriptor such as "2x" or "480w"; a URL containing a literal comma
// has to be percent-encoded under that grammar, so splitting on "," is
// exactly where the spec places the boundary between candidates.
func srcsetCandidateURLs(srcset string) []string {
	var urls []string
	for _, candidate := range strings.Split(srcset, ",") {
		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			continue
		}
		urls = append(urls, fields[0])
	}
	return urls
}

func checkCSSTree(t *testing.T, tree fs.FS, root string) {
	t.Helper()

	found := false
	err := fs.WalkDir(tree, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".css") {
			return nil
		}
		found = true
		data, err := fs.ReadFile(tree, p)
		if err != nil {
			return err
		}
		checkCSS(t, p, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir(%q) error = %v", root, err)
	}
	if !found {
		t.Fatalf("%q has no .css file; run task web:build first", root)
	}
}

// checkCSS fails the test for every absolute @import or url() in css
// that points off this site's own origin. path identifies the CSS in a
// failure message: a file path for a standalone stylesheet, or the HTML
// document's own path for an inline <style> block.
func checkCSS(t *testing.T, path, css string) {
	t.Helper()

	for _, m := range cssExternal.FindAllStringSubmatch(css, -1) {
		url := m[1]
		if url == "" {
			url = m[2]
		}
		t.Errorf("%s: %q references a third party URL %q", path, m[0], url)
	}
}

// isThirdParty reports whether url is an absolute http(s) URL: every
// resource this site serves itself is referenced by a path relative to
// its own origin.
func isThirdParty(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func TestSrcsetCandidateURLs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single candidate with no descriptor",
			input: "/img/logo.png",
			want:  []string{"/img/logo.png"},
		},
		{
			name:  "width descriptors on several candidates",
			input: "/img/logo-480.png 480w, /img/logo-800.png 800w",
			want:  []string{"/img/logo-480.png", "/img/logo-800.png"},
		},
		{
			name:  "density descriptors with extra whitespace",
			input: "  /img/logo.png 1x ,\t/img/logo-2x.png   2x  ",
			want:  []string{"/img/logo.png", "/img/logo-2x.png"},
		},
		{
			name:  "one candidate same origin, one third party",
			input: "/img/logo.png 1x, https://cdn.example.com/logo-2x.png 2x",
			want:  []string{"/img/logo.png", "https://cdn.example.com/logo-2x.png"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "trailing comma",
			input: "/img/logo.png 1x,",
			want:  []string{"/img/logo.png"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := srcsetCandidateURLs(c.input)
			if !slices.Equal(got, c.want) {
				t.Errorf("srcsetCandidateURLs(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}
