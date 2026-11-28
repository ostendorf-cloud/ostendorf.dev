package webapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/ostendorf-cloud/cv/client"
)

// fakeSource answers Get from fixed values and records the language asked.
type fakeSource struct {
	doc  *client.Document
	err  error
	lang string
}

func (f *fakeSource) Get(_ context.Context, lang string) (*client.Document, error) {
	f.lang = lang
	return f.doc, f.err
}

func unavailableBuild() fstest.MapFS {
	_, prerendered := fakeBuild()
	prerendered["unavailable.html"] = &fstest.MapFile{Data: []byte("<title>unavailable</title>")}
	return prerendered
}

func renderProjects(w http.ResponseWriter, _ *http.Request, doc *client.Document) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("projects: " + doc.Projects[0].ID))
}

func TestPageServer_dataHandler(t *testing.T) {
	t.Parallel()

	good := &client.Document{Projects: []client.Project{{ID: "cv"}}}

	cases := []struct {
		name        string
		content     DocumentSource
		prerendered fstest.MapFS
		wantStatus  int
		wantBody    string
		wantRetry   string
	}{
		{
			name:        "a copy exists",
			content:     &fakeSource{doc: good},
			prerendered: unavailableBuild(),
			wantStatus:  http.StatusOK,
			wantBody:    "projects: cv",
		},
		{
			name:        "no copy exists yet",
			content:     &fakeSource{err: errors.New("no copy")},
			prerendered: unavailableBuild(),
			wantStatus:  http.StatusServiceUnavailable,
			wantBody:    "<title>unavailable</title>",
			wantRetry:   "30",
		},
		{
			name:        "cvd is not configured",
			content:     nil,
			prerendered: unavailableBuild(),
			wantStatus:  http.StatusServiceUnavailable,
			wantBody:    "<title>unavailable</title>",
			wantRetry:   "30",
		},
		{
			name:        "the source returns neither a copy nor an error",
			content:     &fakeSource{},
			prerendered: unavailableBuild(),
			wantStatus:  http.StatusServiceUnavailable,
			wantBody:    "<title>unavailable</title>",
			wantRetry:   "30",
		},
		{
			name:        "the placeholder file is missing from the build",
			content:     nil,
			prerendered: func() fstest.MapFS { _, p := fakeBuild(); return p }(),
			wantStatus:  http.StatusServiceUnavailable,
			wantBody:    "503 service unavailable\n",
			wantRetry:   "30",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dist, _ := fakeBuild()
			p, err := newPageServer(dist, tc.prerendered)
			if err != nil {
				t.Fatalf("newPageServer() error = %v", err)
			}
			p.content = tc.content

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/projects/", nil)
			p.dataHandler("en", renderProjects).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if rec.Body.String() != tc.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tc.wantBody)
			}
			if got := rec.Header().Get("Retry-After"); got != tc.wantRetry {
				t.Errorf("Retry-After = %q, want %q", got, tc.wantRetry)
			}
		})
	}
}

func TestPageServer_dataHandler_asksForItsOwnLanguage(t *testing.T) {
	t.Parallel()

	dist, _ := fakeBuild()
	p, err := newPageServer(dist, unavailableBuild())
	if err != nil {
		t.Fatalf("newPageServer() error = %v", err)
	}
	src := &fakeSource{doc: &client.Document{Projects: []client.Project{{ID: "cv"}}}}
	p.content = src

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/projects/", nil)
	p.dataHandler("de", renderProjects).ServeHTTP(rec, req)

	if src.lang != "de" {
		t.Errorf("language asked = %q, want de", src.lang)
	}
}
