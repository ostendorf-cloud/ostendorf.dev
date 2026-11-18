package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const fakeToken = "fake-token-0123456789-not-a-credential"

const sampleDir = "../../internal/content/testdata/artefact"

// TestMain skips the retry backoff: the failing GitHub cases would otherwise
// sit through 7 real seconds each. The schedule itself is tested in
// internal/content with a recording sleep.
func TestMain(m *testing.M) {
	sleepFn = func(context.Context, time.Duration) error { return nil }
	os.Exit(m.Run())
}

// pipeline serves the release assets and the GitHub API from one fake server.
func pipeline(t *testing.T, githubOK bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ostendorf-cloud/cv/releases/download/v1.0.0/", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(filepath.Join(sampleDir, filepath.Base(r.URL.Path)))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if !githubOK {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		switch strings.TrimPrefix(r.URL.Path, "/api") {
		case "/repos/example/repo":
			fmt.Fprint(w, `{"full_name":"example/repo","html_url":"https://github.com/example/repo","description":"d","stargazers_count":1,"forks_count":0,"language":"Go","topics":[],"license":null,"archived":false,"private":false,"pushed_at":"2026-11-01T08:30:00Z"}`)
		case "/repos/example/repo/languages":
			fmt.Fprint(w, `{"Go":10}`)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func sampleDigest(t *testing.T, lang string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sampleDir, "cv."+lang+".json"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// pinned is the environment cv-release.env would give for the sample, plus extra.
func pinned(t *testing.T, extra map[string]string) map[string]string {
	t.Helper()
	m := map[string]string{
		"CV_RELEASE_TAG":       "v1.0.0",
		"CV_RELEASE_SHA256_DE": sampleDigest(t, "de"),
		"CV_RELEASE_SHA256_EN": sampleDigest(t, "en"),
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestRun(t *testing.T) {
	t.Parallel()
	srv := pipeline(t, true)
	out := filepath.Join(t.TempDir(), "gen", "dataset.json")
	var stderr bytes.Buffer
	args := []string{"-out", out, "-cv-base", srv.URL, "-github-api", srv.URL + "/api"}
	e := env(pinned(t, map[string]string{"GITHUB_TOKEN": fakeToken, "SOURCE_DATE_EPOCH": "1795000000"}))

	if err := run(context.Background(), args, e, &stderr); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), args, e, &stderr); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(out)
	if !bytes.Equal(first, second) {
		t.Error("two runs with the same inputs differ")
	}
	if bytes.Contains(first, []byte(fakeToken)) || strings.Contains(stderr.String(), fakeToken) {
		t.Error("token leaked")
	}
	if !bytes.Contains(first, []byte(`"release": "v1.0.0"`)) || !bytes.Contains(first, []byte(`"stars": 1`)) {
		t.Errorf("unexpected dataset:\n%s", first)
	}
}

func TestRunFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		githubOK bool
		env      map[string]string
		extra    []string
		wantErr  string
	}{
		{name: "no pin", githubOK: true, env: map[string]string{}, wantErr: "no cv release is pinned"},
		{name: "moving pin", githubOK: true, env: map[string]string{"CV_RELEASE_TAG": "main"}, wantErr: "moving reference"},
		{name: "release does not exist", githubOK: true, env: map[string]string{"CV_RELEASE_TAG": "v2.0.0"}, wantErr: "status 404"},
		{name: "github down fails the build", githubOK: false, env: pinned(t, map[string]string{"GITHUB_TOKEN": fakeToken}), wantErr: "giving up"},
		{name: "empty digest", githubOK: true, env: pinned(t, map[string]string{"CV_RELEASE_SHA256_EN": ""}), wantErr: "CV_RELEASE_SHA256_EN is empty"},
		{name: "wrong digest", githubOK: true, env: pinned(t, map[string]string{"CV_RELEASE_SHA256_DE": strings.Repeat("0", 64)}), wantErr: "does not match CV_RELEASE_SHA256_DE"},
		{
			name: "stale allowed without a previous file", githubOK: false, env: pinned(t, nil),
			extra: []string{"-allow-stale", "-previous", "/nonexistent/dataset.json"}, wantErr: "no previous dataset",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := pipeline(t, tc.githubOK)
			out := filepath.Join(t.TempDir(), "dataset.json")
			args := append([]string{"-out", out, "-cv-base", srv.URL, "-github-api", srv.URL + "/api"}, tc.extra...)
			var stderr bytes.Buffer
			err := run(context.Background(), args, env(tc.env), &stderr)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %v, want %q", err, tc.wantErr)
			}
			if _, statErr := os.Stat(out); statErr == nil {
				t.Error("a failed run wrote a dataset")
			}
			if strings.Contains(err.Error(), fakeToken) || strings.Contains(stderr.String(), fakeToken) {
				t.Error("token leaked")
			}
		})
	}
}

func TestRunFromArtefactDirRecordsLocal(t *testing.T) {
	t.Parallel()
	srv := pipeline(t, true)
	out := filepath.Join(t.TempDir(), "dataset.json")
	args := []string{"-out", out, "-artefact-dir", sampleDir, "-github-api", srv.URL + "/api"}
	// No pin at all: a local run needs none.
	if err := run(context.Background(), args, env(map[string]string{}), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"release": "local"`)) {
		t.Errorf("provenance not marked local:\n%s", data)
	}
}
