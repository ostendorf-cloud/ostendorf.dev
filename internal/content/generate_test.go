package content

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func generate(t *testing.T, f *fakeGitHub, mutate func(*Options)) ([]byte, error) {
	t.Helper()
	c, _, _ := newClient(t, f, fakeToken)
	o := Options{
		Tag:       "v1.0.0",
		Artefacts: sampleArtefacts(t),
		GitHub:    c,
		Now:       func() time.Time { return fixedNow },
	}
	if mutate != nil {
		mutate(&o)
	}
	if o.Digests == nil {
		// The artefacts as the test left them are the pinned ones, so a test of
		// a parse failure reaches the parser.
		o.Digests = digestsOf(o.Artefacts)
	}
	return Generate(context.Background(), o)
}

func TestGenerateGolden(t *testing.T) {
	t.Parallel()
	got, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "dataset.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if want := readFile(t, golden); !bytes.Equal(got, want) {
		t.Errorf("dataset differs from %s; rerun with -update after checking the diff\n%s", golden, got)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	t.Parallel()
	a, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("same inputs and clock gave different bytes")
	}
	c, err := generate(t, &fakeGitHub{}, func(o *Options) { o.Now = func() time.Time { return fixedNow.Add(time.Hour) } })
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, c) {
		t.Error("the clock does not reach the output")
	}
}

func TestGenerateOutputRoundTrips(t *testing.T) {
	t.Parallel()
	out, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := Load(fstest.MapFS{"generated/" + DatasetFile: {Data: out}})
	if err != nil {
		t.Fatal(err)
	}
	if ds.Source.CV.Release != "v1.0.0" || ds.Source.GitHub.Stale || ds.CV["en"].Projects[0].GitHub.Stars != 42 {
		t.Errorf("unexpected %+v", ds.Source)
	}
	if len(ds.Source.CV.SHA256["en"]) != 64 {
		t.Errorf("digest %q", ds.Source.CV.SHA256["en"])
	}
}

func TestGenerateNeverWritesTheToken(t *testing.T) {
	t.Parallel()
	out, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte(fakeToken)) || bytes.Contains(out, []byte("Bearer")) {
		t.Error("dataset carries the token")
	}
}

func TestGenerateArtefactFailuresAlwaysFail(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mutate  func(*Options)
		wantErr string
	}{
		{"unknown schema version", func(o *Options) { o.Artefacts["en"] = artefact("2", "en", "") }, "unknown schema_version 2"},
		{"malformed artefact", func(o *Options) { o.Artefacts["de"] = []byte("{") }, "not valid JSON"},
		{"missing artefact", func(o *Options) { delete(o.Artefacts, "de") }, "cv artefact de: missing"},
		{"moving reference as the pin", func(o *Options) { o.Tag = "latest" }, "moving reference"},
		{"no pin", func(o *Options) { o.Tag = "" }, "no cv release is pinned"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeGitHub{}
			// Even with a previous dataset and AllowStale, an artefact failure fails.
			_, err := generate(t, f, func(o *Options) {
				o.AllowStale = true
				o.Previous = []byte("irrelevant")
				tc.mutate(o)
			})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %v, want %q", err, tc.wantErr)
			}
			if n := len(f.requested()); n != 0 {
				t.Errorf("GitHub was called %d times before the artefact check", n)
			}
		})
	}
}

func TestGenerateOnlyRequestsRepositoriesTheArtefactNames(t *testing.T) {
	t.Parallel()
	f := &fakeGitHub{}
	if _, err := generate(t, f, nil); err != nil {
		t.Fatal(err)
	}
	for _, p := range f.requested() {
		if !strings.HasPrefix(p, "/repos/example/repo") {
			t.Errorf("request %s", p)
		}
	}
	if len(f.requested()) != 3 {
		t.Errorf("requests %v", f.requested())
	}
}

func failing() *fakeGitHub {
	return &fakeGitHub{override: func(w http.ResponseWriter, _ *http.Request) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	}}
}

func TestFailurePolicy(t *testing.T) {
	t.Parallel()
	good, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	later := func(o *Options) { o.Now = func() time.Time { return fixedNow.Add(48 * time.Hour) } }

	cases := []struct {
		name      string
		mutate    func(*Options)
		wantErr   string
		wantStale bool
	}{
		{name: "default: a failed fetch fails the build", mutate: func(o *Options) { o.Previous = good }, wantErr: "giving up"},
		{name: "allow stale without a previous dataset fails", mutate: func(o *Options) { o.AllowStale = true }, wantErr: "no previous dataset given"},
		{
			name: "allow stale with a good previous dataset ships it stale", wantStale: true,
			mutate: func(o *Options) { o.AllowStale = true; o.Previous = good; later(o) },
		},
		{
			name: "previous dataset too old fails", wantErr: "old, the limit is",
			mutate: func(o *Options) {
				o.AllowStale = true
				o.Previous = good
				o.MaxStaleAge = time.Hour
				later(o)
			},
		},
		{
			name: "previous dataset is garbage", wantErr: "previous dataset",
			mutate: func(o *Options) { o.AllowStale = true; o.Previous = []byte("{}") },
		},
		{
			name: "previous dataset misses a repository fails", wantErr: "has no data for example/repo",
			mutate: func(o *Options) {
				o.AllowStale = true
				o.Previous = bytes.ReplaceAll(good, []byte(`"github_repo": "example/repo"`), []byte(`"github_repo": "example/else"`))
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := generate(t, failing(), tc.mutate)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %v, want %q", err, tc.wantErr)
				}
				if out != nil {
					t.Error("a failed build returned data")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			ds, err := ParseDataset(out)
			if err != nil {
				t.Fatal(err)
			}
			gh := ds.Source.GitHub
			if gh.Stale != tc.wantStale || gh.StaleReason == nil || !strings.Contains(*gh.StaleReason, "github fetch failed") {
				t.Errorf("source %+v", gh)
			}
			// The stale data keeps its own fetch time, not the rebuild's.
			if gh.FetchedAt != fixedNow.Format(time.RFC3339) {
				t.Errorf("fetched_at %s", gh.FetchedAt)
			}
			if strings.Contains(string(out), fakeToken) {
				t.Error("token in output")
			}
		})
	}
}

func TestFreshFetchIgnoresPrevious(t *testing.T) {
	t.Parallel()
	out, err := generate(t, &fakeGitHub{}, func(o *Options) { o.AllowStale = true; o.Previous = []byte("garbage") })
	if err != nil {
		t.Fatal(err)
	}
	ds, err := ParseDataset(out)
	if err != nil || ds.Source.GitHub.Stale || ds.Source.GitHub.StaleReason != nil {
		t.Errorf("%v %+v", err, ds.Source.GitHub)
	}
}

func TestParseDatasetStrict(t *testing.T) {
	t.Parallel()
	good, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{"unknown key", bytes.Replace(good, []byte(`"stale": false`), []byte(`"stale": false, "extra": 1`), 1), "unknown field"},
		{"wrong version", bytes.Replace(good, []byte(`"schema_version": 1,`), []byte(`"schema_version": 9,`), 1), "schema_version 9"},
		{"project without github data", bytes.Replace(good, []byte(`"github": {`), []byte(`"github_x": {`), 1), "unknown field"},
		{"stale flag without reason", bytes.Replace(good, []byte(`"stale": false`), []byte(`"stale": true`), 1), "disagree"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseDataset(tc.data); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestDatasetValidateCatchesAnUnmergedProject(t *testing.T) {
	t.Parallel()
	good, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := ParseDataset(good)
	if err != nil {
		t.Fatal(err)
	}
	ds.CV["en"].Projects[0].GitHub = nil
	if err := ds.Validate(); err == nil || !strings.Contains(err.Error(), "has no GitHub data") {
		t.Errorf("error %v", err)
	}
	delete(ds.CV, "de")
	if err := ds.Validate(); err == nil || !strings.Contains(err.Error(), `no document for language "de"`) {
		t.Errorf("error %v", err)
	}
}

func TestValidateTag(t *testing.T) {
	t.Parallel()
	for tag, ok := range map[string]bool{
		"v1.0.0": true, "v0.12.3": true, "v1.0.0-rc.1": true,
		"": false, "latest": false, "main": false, "1.0.0": false, "v1": false, "v1.0.0/../x": false, "v1.0.0?x": false,
	} {
		if err := ValidateTag(tag); (err == nil) != ok {
			t.Errorf("%q: %v", tag, err)
		}
	}
}

func TestDownloadArtefacts(t *testing.T) {
	t.Parallel()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Authorization") != "" {
			t.Error("artefact download carries credentials")
		}
		data, err := os.ReadFile(filepath.Join("testdata", "artefact", filepath.Base(r.URL.Path)))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	got, err := DownloadArtefacts(context.Background(), srv.Client(), srv.URL, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !bytes.Equal(got["en"], sampleArtefacts(t)["en"]) {
		t.Error("wrong content")
	}
	want := []string{
		"/ostendorf-cloud/cv/releases/download/v1.0.0/cv.de.json",
		"/ostendorf-cloud/cv/releases/download/v1.0.0/cv.en.json",
	}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Errorf("paths %v", paths)
	}

	missing := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(missing.Close)
	if _, err := DownloadArtefacts(context.Background(), missing.Client(), missing.URL, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Errorf("error %v", err)
	}
	if _, err := DownloadArtefacts(context.Background(), nil, "", "main"); err == nil {
		t.Error("a moving reference was accepted")
	}
}

func TestErrorsDoNotCarryTheToken(t *testing.T) {
	t.Parallel()
	_, err := generate(t, failing(), nil)
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Error("token in error")
	}
	var rl *RateLimitError
	if errors.As(err, &rl) {
		t.Error("a 500 is not a rate limit")
	}
}
