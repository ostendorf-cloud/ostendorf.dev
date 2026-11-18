package content

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateRepo(t *testing.T) {
	t.Parallel()
	for name, ok := range map[string]bool{
		"example/repo": true, "a/b": true, "my-org/my.repo_1": true, "a1-b2/.github": true,
		"../users": false, "./x": false, "owner/..": false, "owner/.": false, "../..": false,
		"-a/b": false, "a-/b": false, "a--b/c": false, "a_b/c": false, "a/b/c": false,
		"a/b c": false, "a/b?x": false, "a/b#x": false, "a/%2e%2e": false, `a/b\c`: false, "": false, "a": false, "/b": false, "a/": false,
		strings.Repeat("a", 40) + "/b": false, "a/" + strings.Repeat("b", 101): false,
	} {
		if err := ValidateRepo(name); (err == nil) != ok {
			t.Errorf("%q: %v", name, err)
		}
	}
}

func TestCheckDigest(t *testing.T) {
	t.Parallel()
	raw := []byte("artefact bytes")
	want := digestsOf(map[string][]byte{"en": raw})["en"]
	cases := []struct {
		name    string
		pin     string
		wantErr string
	}{
		{name: "match", pin: want},
		{name: "upper case pin matches", pin: strings.ToUpper(want)},
		{name: "mismatch", pin: strings.Repeat("0", 64), wantErr: "does not match CV_RELEASE_SHA256_EN in cv-release.env"},
		{name: "empty", pin: "", wantErr: "CV_RELEASE_SHA256_EN is empty"},
		{name: "odd length", pin: want[:63], wantErr: "is not a sha256"},
		{name: "even but too short", pin: want[:62], wantErr: "is not a sha256"},
		{name: "even but too long", pin: want + "00", wantErr: "is not a sha256"},
		{name: "not hex", pin: strings.Repeat("z", 64), wantErr: "is not a sha256"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := CheckDigest("en", "v1.0.0", raw, tc.pin)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestCheckDigestMismatchNamesFileAndTagAndPrintsTwoHashesOnly(t *testing.T) {
	t.Parallel()
	raw := []byte("secret looking artefact content")
	err := CheckDigest("de", "v1.2.3", raw, strings.Repeat("a", 64))
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{"cv-release.env", "v1.2.3", "cv.de.json", "CV_RELEASE_SHA256_DE", strings.Repeat("a", 64), digestsOf(map[string][]byte{"x": raw})["x"]} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "secret looking") {
		t.Error("error prints artefact content")
	}
}

func TestGenerateDigestPin(t *testing.T) {
	t.Parallel()
	wrong := strings.Repeat("0", 64)
	cases := []struct {
		name    string
		mutate  func(*Options)
		wantErr string
	}{
		{"mismatch of one language", func(o *Options) { o.Digests = digestsOf(o.Artefacts); o.Digests["de"] = wrong }, "cv.de.json of release v1.0.0 does not match"},
		{"empty pin with a set tag", func(o *Options) { o.Digests = map[string]string{"de": digestsOf(o.Artefacts)["de"]} }, "CV_RELEASE_SHA256_EN is empty"},
		{"no digests at all", func(o *Options) { o.Digests = map[string]string{} }, "is empty"},
		{"artefact replaced after the pin", func(o *Options) {
			o.Digests = digestsOf(o.Artefacts)
			o.Artefacts["en"] = append([]byte(nil), o.Artefacts["en"]...)
			o.Artefacts["en"] = bytes.Replace(o.Artefacts["en"], []byte("Engineer en"), []byte("Engineer EN"), 1)
		}, "cv.en.json of release v1.0.0 does not match"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeGitHub{}
			_, err := generate(t, f, tc.mutate)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %v, want %q", err, tc.wantErr)
			}
			if n := len(f.requested()); n != 0 {
				t.Errorf("GitHub called %d times before the digest check", n)
			}
		})
	}
}

func TestGenerateLocalSkipsThePinAndSaysSo(t *testing.T) {
	t.Parallel()
	out, err := generate(t, &fakeGitHub{}, func(o *Options) { o.Local = true; o.Tag = ""; o.Digests = map[string]string{} })
	if err != nil {
		t.Fatal(err)
	}
	ds, err := ParseDataset(out)
	if err != nil {
		t.Fatal(err)
	}
	if ds.Source.CV.Release != "local" {
		t.Errorf("release %q", ds.Source.CV.Release)
	}
	if len(ds.Source.CV.SHA256["en"]) != 64 {
		t.Error("digest of the local file is not recorded")
	}
}

func TestPreviousDatasetAgeComesFromFetchedAtNotGeneratedAt(t *testing.T) {
	t.Parallel()
	good, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := ParseDataset(good)
	if err != nil {
		t.Fatal(err)
	}
	// An earlier stale rebuild: written recently, GitHub data 20 days old.
	reason := "github fetch failed: earlier"
	ds.Source.GitHub = GitHubSource{FetchedAt: fixedNow.Add(-20 * 24 * time.Hour).Format(time.RFC3339), Stale: true, StaleReason: &reason}
	ds.GeneratedAt = fixedNow.Format(time.RFC3339)
	prev, err := ds.Encode()
	if err != nil {
		t.Fatal(err)
	}
	_, err = generate(t, failing(), func(o *Options) { o.AllowStale = true; o.Previous = prev })
	if err == nil || !strings.Contains(err.Error(), "old, the limit is") {
		t.Fatalf("a stale dataset was rebuilt as fresh stale: %v", err)
	}
}

func TestPreviousDatasetFromTheFutureIsRefused(t *testing.T) {
	t.Parallel()
	good, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = generate(t, failing(), func(o *Options) {
		o.AllowStale = true
		o.Previous = good
		o.Now = func() time.Time { return fixedNow.Add(-time.Hour) }
	})
	if err == nil || !strings.Contains(err.Error(), "in the future") {
		t.Fatalf("error %v", err)
	}
}

func TestParseDatasetRejectsTrailingData(t *testing.T) {
	t.Parallel()
	good, err := generate(t, &fakeGitHub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tail := range []string{`{}`, `x`, `[]`} {
		if _, err := ParseDataset(append(append([]byte(nil), good...), tail...)); err == nil || !strings.Contains(err.Error(), "unexpected data") {
			t.Errorf("tail %q: %v", tail, err)
		}
	}
	if _, err := ParseDataset(append(append([]byte(nil), good...), " \n\n"...)); err != nil {
		t.Errorf("trailing whitespace refused: %v", err)
	}
}

func TestGitHubPhaseHasAnOverallDeadline(t *testing.T) {
	t.Parallel()
	f := &fakeGitHub{override: func(_ http.ResponseWriter, r *http.Request) bool {
		<-r.Context().Done()
		return true
	}}
	start := time.Now()
	_, err := generate(t, f, func(o *Options) { o.GitHubTimeout = 50 * time.Millisecond })
	if err == nil || !strings.Contains(err.Error(), "overall deadline of 50ms") {
		t.Fatalf("error %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("the deadline did not end the build")
	}
}

// Tag injection: nothing but a version tag may reach a URL.
func TestValidateTagRejectsInjection(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{
		"v1.0.0-a/../../x", "v1.0.0-x@evil", "v1.0.0-a%2f..", "v1.0.0-a:b", `v1.0.0-a\b`, "v1.0.0-a/b",
		"v1.0.0-a?b", "v1.0.0-a#b", "v1.0.0 ", " v1.0.0", "v1.0.0\n", "v1.0.0-", "v1.0.0-a b",
	} {
		if err := ValidateTag(tag); err == nil {
			t.Errorf("%q accepted", tag)
		}
	}
}

func TestReleaseURLIsBuiltFromEscapedSegments(t *testing.T) {
	t.Parallel()
	got, err := releaseURL("https://github.com/", "v1.0.0-rc.1", "cv.en.json")
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://github.com/ostendorf-cloud/cv/releases/download/v1.0.0-rc.1/cv.en.json"; got != want {
		t.Errorf("got %s", got)
	}
}

func TestReleaseRedirectHostsAreMatchedWhole(t *testing.T) {
	t.Parallel()
	if got := strings.Join(ReleaseRedirectHosts, ","); got != "github.com,objects.githubusercontent.com,release-assets.githubusercontent.com" {
		t.Errorf("hosts %s", got)
	}
	allowed := releaseRedirectAllowed("github.com")
	for raw, ok := range map[string]bool{
		"https://github.com/x":                               true,
		"https://objects.githubusercontent.com/x":            true,
		"https://release-assets.githubusercontent.com/x":     true,
		"https://github.com.evil.example/x":                  false,
		"https://evilgithub.com/x":                           false,
		"https://githubusercontent.com/x":                    false,
		"https://raw.githubusercontent.com/x":                false,
		"https://objects.githubusercontent.com.evil.test/x":  false,
		"https://user@release-assets.githubusercontent.com/": true,
	} {
		u, _ := url.Parse(raw)
		if allowed(u) != ok {
			t.Errorf("%s: allowed=%v, want %v", raw, !ok, ok)
		}
	}
}

func TestRedirectPolicy(t *testing.T) {
	t.Parallel()
	all := func(*url.URL) bool { return true }
	mk := func(raw string) *http.Request {
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, raw, nil)
		return r
	}
	via := func(n int, first string) []*http.Request {
		v := make([]*http.Request, n)
		for i := range v {
			v[i] = mk(first)
		}
		return v
	}
	if err := redirectPolicy(all)(mk("https://a.example/x"), via(5, "https://a.example/")); err != nil {
		t.Errorf("5 hops refused: %v", err)
	}
	if err := redirectPolicy(all)(mk("https://a.example/x"), via(6, "https://a.example/")); err == nil {
		t.Error("6 hops followed")
	}
	if err := redirectPolicy(all)(mk("http://a.example/x"), via(1, "https://a.example/")); err == nil {
		t.Error("http followed")
	}
	if err := redirectPolicy(func(*url.URL) bool { return false })(mk("https://a.example/x"), via(1, "https://a.example/")); err == nil {
		t.Error("a host that is not allowed was followed")
	}
	req := mk("https://b.example/x")
	req.Header.Set("Authorization", "Bearer "+fakeToken)
	if err := redirectPolicy(all)(req, via(1, "https://a.example/")); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "" {
		t.Error("Authorization kept across a host change")
	}
	same := mk("https://a.example/y")
	same.Header.Set("Authorization", "x")
	_ = redirectPolicy(all)(same, via(1, "https://a.example/"))
	if same.Header.Get("Authorization") == "" {
		t.Error("Authorization dropped on the same host")
	}
}

// redirector serves a TLS server that redirects every request to target.
func redirector(t *testing.T, target func() string, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, target()+r.URL.Path, http.StatusMovedPermanently)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAPIClientRedirects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		other     func(t *testing.T, hits *atomic.Int32) string
		wantRefus string
	}{
		{"to plain http", func(t *testing.T, hits *atomic.Int32) string {
			s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
			t.Cleanup(s.Close)
			return s.URL
		}, "only https is followed"},
		{"to another https host", func(t *testing.T, hits *atomic.Int32) string {
			s := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
			t.Cleanup(s.Close)
			return s.URL
		}, "which is not allowed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var otherHits, firstHits atomic.Int32
			var target string
			first := redirector(t, func() string { return target }, &firstHits)
			target = tc.other(t, &otherHits)
			c := &Client{BaseURL: first.URL, HTTP: first.Client(), Token: fakeToken, Sleep: func(context.Context, time.Duration) error { return nil }}
			_, err := c.FetchRepo(context.Background(), "example/repo")
			if err == nil || !strings.Contains(err.Error(), tc.wantRefus) {
				t.Fatalf("error %v", err)
			}
			if otherHits.Load() != 0 {
				t.Error("the redirect target was contacted")
			}
			if firstHits.Load() != 1 {
				t.Errorf("%d requests to the API host, a refused redirect must not retry", firstHits.Load())
			}
			if strings.Contains(err.Error(), fakeToken) {
				t.Error("error carries the token")
			}
		})
	}
}

func TestAPIClientFollowsARedirectOnItsOwnHost(t *testing.T) {
	t.Parallel()
	var auths []string
	h := &fakeGitHub{}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/example/other" && r.URL.Query().Get("r") == "" {
			http.Redirect(w, r, r.URL.Path+"?r=1", http.StatusMovedPermanently)
			return
		}
		auths = append(auths, r.Header.Get("Authorization"))
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: fakeToken}
	if _, err := c.FetchRepo(context.Background(), "example/other"); err != nil {
		t.Fatal(err)
	}
	if len(auths) == 0 || auths[0] != "Bearer "+fakeToken {
		t.Errorf("auths %v", auths)
	}
}

func TestAPIClientStopsAfterFiveRedirects(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, r.URL.Path+"x", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.FetchRepo(context.Background(), "example/repo")
	if err == nil || !strings.Contains(err.Error(), "more than 5 redirects") {
		t.Fatalf("error %v", err)
	}
	if hits.Load() != 6 {
		t.Errorf("%d requests", hits.Load())
	}
}

func TestDownloadRedirects(t *testing.T) {
	t.Parallel()
	sample := sampleArtefacts(t)

	t.Run("to plain http is refused", func(t *testing.T) {
		t.Parallel()
		var otherHits, firstHits atomic.Int32
		plain := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { otherHits.Add(1) }))
		t.Cleanup(plain.Close)
		first := redirector(t, func() string { return plain.URL }, &firstHits)
		_, err := DownloadArtefacts(context.Background(), first.Client(), first.URL, "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "only https is followed") || otherHits.Load() != 0 {
			t.Errorf("error %v, hits %d", err, otherHits.Load())
		}
	})

	t.Run("to another host is refused", func(t *testing.T) {
		t.Parallel()
		var otherHits, firstHits atomic.Int32
		other := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { otherHits.Add(1) }))
		t.Cleanup(other.Close)
		first := redirector(t, func() string { return other.URL }, &firstHits)
		_, err := DownloadArtefacts(context.Background(), first.Client(), first.URL, "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "which is not allowed") || otherHits.Load() != 0 {
			t.Errorf("error %v, hits %d", err, otherHits.Load())
		}
	})

	t.Run("on its own host is followed and no request carries credentials", func(t *testing.T) {
		t.Parallel()
		var seen atomic.Int32
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
				t.Error("a download request carries credentials")
			}
			if r.URL.Query().Get("r") == "" {
				http.Redirect(w, r, r.URL.Path+"?r=1", http.StatusFound)
				return
			}
			seen.Add(1)
			data, err := os.ReadFile(filepath.Join("testdata", "artefact", filepath.Base(r.URL.Path)))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(data)
		}))
		t.Cleanup(srv.Close)
		got, err := DownloadArtefacts(context.Background(), srv.Client(), srv.URL, "v1.0.0")
		if err != nil || !bytes.Equal(got["de"], sample["de"]) || seen.Load() != 2 {
			t.Errorf("error %v, seen %d", err, seen.Load())
		}
	})
}
