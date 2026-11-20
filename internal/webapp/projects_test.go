package webapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/content"
	"github.com/ostendorf-cloud/ostendorf.dev/internal/routes"
)

func datasetWith(de, en []string) content.Dataset {
	doc := func(lang string, ids []string) content.Document {
		d := content.Document{Language: lang}
		for _, id := range ids {
			d.Projects = append(d.Projects, content.Project{ID: id})
		}
		return d
	}
	return content.Dataset{CV: map[string]content.Document{"de": doc("de", de), "en": doc("en", en)}}
}

func TestProjectRoutesAreSortedAndNamedByID(t *testing.T) {
	t.Parallel()

	got, err := ProjectRoutes(datasetWith([]string{"zeta", "alpha-1"}, []string{"alpha-1", "zeta"}))
	if err != nil {
		t.Fatalf("ProjectRoutes() error = %v", err)
	}
	want := []routes.Route{
		{Name: "project:alpha-1", Slug: "projects/alpha-1", File: "projects/alpha-1/index.html"},
		{Name: "project:zeta", Slug: "projects/zeta", File: "projects/zeta/index.html"},
	}
	if len(got) != len(want) {
		t.Fatalf("ProjectRoutes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("route %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestProjectRoutesRejectAnUnsafeID(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"", "Upper", "a/b", "..", "a b", "-a", "a--b", "a?b"} {
		if _, err := ProjectRoutes(datasetWith([]string{id}, []string{id})); err == nil {
			t.Errorf("ProjectRoutes() accepted the id %q", id)
		}
	}
}

func TestProjectRoutesRejectLanguagesThatDisagree(t *testing.T) {
	t.Parallel()

	if _, err := ProjectRoutes(datasetWith([]string{"a"}, []string{"a", "b"})); err == nil {
		t.Fatal("ProjectRoutes() error = nil, want an error for differing project sets")
	}
}

func TestHandlerServesAndListsTheProjectPages(t *testing.T) {
	t.Parallel()

	extra, err := ProjectRoutes(datasetWith([]string{"demo"}, []string{"demo"}))
	if err != nil {
		t.Fatalf("ProjectRoutes() error = %v", err)
	}
	dist, prerendered := fakeBuild()
	prerendered["projects/demo/index.html"] = &fstest.MapFile{Data: []byte("<title>demo</title>")}
	cfg := testConfig()
	cfg.Routes = extra
	h, err := NewHandler(cfg, dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		return rec
	}

	if rec := get("/projects/demo/"); rec.Code != http.StatusOK || rec.Body.String() != "<title>demo</title>" {
		t.Errorf("GET /projects/demo/ = %d %q, want the prerendered project page", rec.Code, rec.Body.String())
	}
	if rec := get("/projects/demo"); rec.Header().Get("Location") != "/projects/demo/" {
		t.Errorf("GET /projects/demo = %d to %q, want a redirect to the slash form",
			rec.Code, rec.Header().Get("Location"))
	}
	if rec := get("/projects/other/"); rec.Code != http.StatusNotFound {
		t.Errorf("GET /projects/other/ = %d, want 404", rec.Code)
	}
	if body := get("/sitemap.xml").Body.String(); !strings.Contains(body, "https://ostendorf.dev/projects/demo/") {
		t.Errorf("sitemap.xml = %q, want the project page listed", body)
	}
}

func TestNewHandlerRejectsAnExtraRouteThatRepeatsASlug(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	cfg := testConfig()
	cfg.Routes = []routes.Route{{Name: "x", Slug: "projects", File: "x.html"}}
	if _, err := NewHandler(cfg, dist, prerendered); err == nil {
		t.Fatal("NewHandler() error = nil, want an error for a repeated slug")
	}
}
