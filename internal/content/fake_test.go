package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fakeToken is a made up string, never a real credential. The tests assert it
// appears in no error, log line or output.
const fakeToken = "fake-token-0123456789-not-a-credential"

var fixedNow = time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

// fakeGitHub answers the three endpoints the client uses for the repositories
// it knows, and records every path it was asked for.
type fakeGitHub struct {
	mu       sync.Mutex
	paths    []string
	auths    []string
	override func(w http.ResponseWriter, r *http.Request) bool
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	f.auths = append(f.auths, r.Header.Get("Authorization"))
	f.mu.Unlock()
	if f.override != nil && f.override(w, r) {
		return
	}
	switch r.URL.Path {
	case "/repos/example/repo":
		fmt.Fprint(w, `{"full_name":"example/repo","html_url":"https://github.com/example/repo",
"description":"A fake repository.","stargazers_count":42,"forks_count":3,"language":"Go",
"topics":["zeta","alpha"],"license":{"spdx_id":"Apache-2.0"},"archived":false,"private":false,
"pushed_at":"2026-11-01T08:30:00Z"}`)
	case "/repos/example/repo/languages":
		fmt.Fprint(w, `{"Shell":1000,"Vue":9000,"Go":9000}`)
	case "/repos/example/repo/releases/latest":
		fmt.Fprint(w, `{"tag_name":"v1.2.3","html_url":"https://github.com/example/repo/releases/tag/v1.2.3","published_at":"2026-10-20T12:00:00Z"}`)
	case "/repos/example/other":
		fmt.Fprint(w, `{"full_name":"example/other","html_url":"https://github.com/example/other",
"description":null,"stargazers_count":0,"forks_count":0,"language":null,
"topics":[],"license":{"spdx_id":"NOASSERTION"},"archived":true,"private":false,
"pushed_at":"2025-01-01T00:00:00Z"}`)
	case "/repos/example/other/languages":
		fmt.Fprint(w, `{}`)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeGitHub) requested() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

// newClient starts the fake and returns a client for it. Sleeping is recorded,
// not done.
func newClient(t *testing.T, f *fakeGitHub, token string) (*Client, *[]time.Duration, *bytes.Buffer) {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	var sleeps []time.Duration
	logs := &bytes.Buffer{}
	return &Client{
		BaseURL: srv.URL,
		HTTP:    srv.Client(),
		Token:   token,
		Now:     func() time.Time { return fixedNow },
		Sleep: func(_ context.Context, d time.Duration) error {
			sleeps = append(sleeps, d)
			return nil
		},
		Log: slog.New(slog.NewTextHandler(logs, nil)),
	}, &sleeps, logs
}

func sampleArtefacts(t *testing.T) map[string][]byte {
	t.Helper()
	m, err := ReadArtefacts(filepath.Join("testdata", "artefact"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func digestsOf(artefacts map[string][]byte) map[string]string {
	out := map[string]string{}
	for lang, raw := range artefacts {
		sum := sha256.Sum256(raw)
		out[lang] = hex.EncodeToString(sum[:])
	}
	return out
}

func readFile(t *testing.T, parts ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
