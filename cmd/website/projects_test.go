package main

import (
	"html"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/content"
)

var (
	titlePattern = regexp.MustCompile(`<title>([^<]*)</title>`)
	descPattern  = regexp.MustCompile(`<meta name="description" content="([^"]*)">`)
)

// TestPrerenderedProjectPagesAreReadableWithoutJavaScript reads the real
// output "task prerender" wrote for every project of the embedded dataset. The
// bytes below are all a visitor with JavaScript switched off, or a crawler
// that does not run it, ever receives: the description, the languages and the
// release have to be in them, and each page needs its own title and
// description.
func TestPrerenderedProjectPagesAreReadableWithoutJavaScript(t *testing.T) {
	dataset, err := content.Embedded()
	if err != nil {
		t.Fatalf("content.Embedded() error = %v", err)
	}
	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub() error = %v", err)
	}
	index, err := fs.ReadFile(sub, "projects/index.html")
	if err != nil {
		t.Fatalf("the project index is not prerendered: %v", err)
	}

	projects := dataset.CV["en"].Projects
	if len(projects) == 0 {
		t.Fatal("the embedded dataset has no projects")
	}
	titles := map[string]string{}
	descriptions := map[string]string{}
	for _, p := range projects {
		t.Run(p.ID, func(t *testing.T) {
			data, err := fs.ReadFile(sub, "projects/"+p.ID+"/index.html")
			if err != nil {
				t.Fatalf("project %q has no prerendered page: %v", p.ID, err)
			}
			page := string(data)

			if !strings.Contains(string(index), `href="/projects/`+p.ID+`/"`) {
				t.Errorf("the index does not link to /projects/%s/", p.ID)
			}
			if desc := strings.TrimSpace(p.Description); desc != "" {
				if !strings.Contains(page, html.EscapeString(desc)) {
					t.Errorf("page lacks the description %q", desc)
				}
				if !strings.Contains(string(index), html.EscapeString(desc)) {
					t.Errorf("index lacks the description %q", desc)
				}
			}
			if len(p.GitHub.Languages) > 0 && !strings.Contains(page, p.GitHub.Languages[0].Name) {
				t.Errorf("page lacks the language %q", p.GitHub.Languages[0].Name)
			}
			if rel := p.GitHub.LatestRelease; rel != nil && !strings.Contains(page, rel.Tag) {
				t.Errorf("page lacks the release %q", rel.Tag)
			}

			title := titlePattern.FindStringSubmatch(page)
			desc := descPattern.FindStringSubmatch(page)
			if title == nil || desc == nil {
				t.Fatalf("page lacks a title or a meta description")
			}
			if other, dup := titles[title[1]]; dup {
				t.Errorf("title %q is also the title of %s", title[1], other)
			}
			titles[title[1]] = p.ID
			if other, dup := descriptions[desc[1]]; dup {
				t.Errorf("meta description %q is also the one of %s", desc[1], other)
			}
			descriptions[desc[1]] = p.ID
		})
	}
}
