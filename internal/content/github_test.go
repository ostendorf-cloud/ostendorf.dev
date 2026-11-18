package content

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFetchRepoSuccess(t *testing.T) {
	t.Parallel()
	c, sleeps, _ := newClient(t, &fakeGitHub{}, "")
	got, err := c.FetchRepo(context.Background(), "example/repo")
	if err != nil {
		t.Fatal(err)
	}
	desc, lang, lic := "A fake repository.", "Go", "Apache-2.0"
	want := &Repo{
		FullName: "example/repo", URL: "https://github.com/example/repo", Description: &desc,
		Stars: 42, Forks: 3, Language: &lang,
		// Equal bytes sort by name, so Go comes before Vue.
		Languages: []LanguageShare{{"Go", 9000}, {"Vue", 9000}, {"Shell", 1000}},
		Topics:    []string{"alpha", "zeta"}, License: &lic, PushedAt: "2026-11-01T08:30:00Z",
		LatestRelease: &Release{Tag: "v1.2.3", URL: "https://github.com/example/repo/releases/tag/v1.2.3", PublishedAt: "2026-10-20T12:00:00Z"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if len(*sleeps) != 0 {
		t.Errorf("slept %v on a clean run", *sleeps)
	}
}

func TestFetchRepoNoReleaseAndNoLicense(t *testing.T) {
	t.Parallel()
	c, _, _ := newClient(t, &fakeGitHub{}, "")
	got, err := c.FetchRepo(context.Background(), "example/other")
	if err != nil {
		t.Fatal(err)
	}
	if got.LatestRelease != nil || got.License != nil || got.Description != nil || !got.Archived {
		t.Errorf("unexpected %+v", got)
	}
	if got.Languages == nil || got.Topics == nil {
		t.Error("lists must be empty, not null")
	}
}

func TestFetchRepoFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		repo       string
		override   func(w http.ResponseWriter, r *http.Request) bool
		wantErr    string
		wantIs     error
		wantAsRate bool
		wantReqs   int
		wantSleeps []time.Duration
	}{
		{name: "unknown repository", repo: "example/missing", wantErr: "not found", wantIs: ErrNotFound, wantReqs: 1},
		{
			name: "private repository", repo: "example/repo", wantErr: "private", wantReqs: 1,
			override: func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path == "/repos/example/repo" {
					fmt.Fprint(w, `{"full_name":"example/repo","private":true,"pushed_at":"2026-01-01T00:00:00Z"}`)
					return true
				}
				return false
			},
		},
		{
			name: "403 with a short Retry-After then success", repo: "example/other", wantReqs: 4, wantSleeps: []time.Duration{5 * time.Second},
			override: func() func(w http.ResponseWriter, r *http.Request) bool {
				n := 0
				return func(w http.ResponseWriter, r *http.Request) bool {
					n++
					if n == 1 {
						w.Header().Set("Retry-After", "5")
						w.WriteHeader(http.StatusForbidden)
						return true
					}
					return false
				}
			}(),
		},
		{
			name: "429 with a Retry-After beyond the limit", repo: "example/repo", wantAsRate: true, wantReqs: 1,
			override: func(w http.ResponseWriter, _ *http.Request) bool {
				w.Header().Set("Retry-After", "3600")
				w.WriteHeader(http.StatusTooManyRequests)
				return true
			},
		},
		{
			name: "403 with an exhausted quota beyond the limit", repo: "example/repo", wantAsRate: true, wantReqs: 1,
			override: func(w http.ResponseWriter, _ *http.Request) bool {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(fixedNow.Add(time.Hour).Unix(), 10))
				w.WriteHeader(http.StatusForbidden)
				return true
			},
		},
		{
			name: "403 without a rate limit signal is final", repo: "example/repo", wantErr: "status 403", wantReqs: 1,
			override: func(w http.ResponseWriter, _ *http.Request) bool {
				w.WriteHeader(http.StatusForbidden)
				return true
			},
		},
		{
			name: "5xx on every attempt", repo: "example/repo", wantErr: "giving up after 4 attempts", wantReqs: 4,
			wantSleeps: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
			override: func(w http.ResponseWriter, _ *http.Request) bool {
				w.WriteHeader(http.StatusBadGateway)
				return true
			},
		},
		{
			name: "malformed body", repo: "example/repo", wantErr: "malformed response", wantReqs: 1,
			override: func(w http.ResponseWriter, _ *http.Request) bool {
				fmt.Fprint(w, `{"full_name": `)
				return true
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeGitHub{override: tc.override}
			c, sleeps, logs := newClient(t, f, fakeToken)
			_, err := c.FetchRepo(context.Background(), tc.repo)
			if tc.name == "403 with a short Retry-After then success" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("want an error")
			}
			if tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q lacks %q", err, tc.wantErr)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Errorf("error %v is not %v", err, tc.wantIs)
			}
			var rl *RateLimitError
			if tc.wantAsRate != errors.As(err, &rl) {
				t.Errorf("rate limit error: got %v, want %v (err %v)", !tc.wantAsRate, tc.wantAsRate, err)
			}
			if got := len(f.requested()); tc.wantReqs != 0 && got != tc.wantReqs {
				t.Errorf("%d requests, want %d: %v", got, tc.wantReqs, f.requested())
			}
			if tc.wantSleeps != nil && !reflect.DeepEqual(*sleeps, tc.wantSleeps) {
				t.Errorf("sleeps %v, want %v", *sleeps, tc.wantSleeps)
			}
			if err != nil && strings.Contains(err.Error(), fakeToken) {
				t.Error("error carries the token")
			}
			if strings.Contains(logs.String(), fakeToken) {
				t.Error("log carries the token")
			}
		})
	}
}

func TestRateLimitResetWaitIsHonoured(t *testing.T) {
	t.Parallel()
	n := 0
	f := &fakeGitHub{override: func(w http.ResponseWriter, _ *http.Request) bool {
		n++
		if n == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(fixedNow.Add(30*time.Second).Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
			return true
		}
		return false
	}}
	c, sleeps, _ := newClient(t, f, "")
	if _, err := c.FetchRepo(context.Background(), "example/other"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*sleeps, []time.Duration{30 * time.Second}) {
		t.Errorf("sleeps %v", *sleeps)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchRepoTimeoutRetriesThenFails(t *testing.T) {
	t.Parallel()
	calls := 0
	c := &Client{
		BaseURL: "https://api.invalid",
		HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, context.DeadlineExceeded
		})},
		Token: fakeToken,
		Sleep: func(context.Context, time.Duration) error { return nil },
	}
	_, err := c.FetchRepo(context.Background(), "example/repo")
	if err == nil || !strings.Contains(err.Error(), "giving up after 4 attempts") {
		t.Fatalf("got %v", err)
	}
	if calls != 4 {
		t.Errorf("%d calls, want 4", calls)
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Error("error carries the token")
	}
}

func TestFetchRepoRejectsBadName(t *testing.T) {
	t.Parallel()
	f := &fakeGitHub{}
	c, _, _ := newClient(t, f, "")
	for _, name := range []string{
		"", "example", "a/b/c", "../x/y", "a b/c", "a/b?x=1",
		"../users", "./x", "owner/..", "owner/.", "../..", "-a/b", "a-/b", "a--b/c", "a/b#f", "a/b%2e", `a\b/c`,
	} {
		if _, err := c.FetchRepo(context.Background(), name); err == nil {
			t.Errorf("%q accepted", name)
		}
	}
	if got := f.requested(); len(got) != 0 {
		t.Errorf("a rejected name still caused requests: %v", got)
	}
}

func TestFetchRepoRejectsAResponseForAnotherRepository(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, fullName string
		wantErr        bool
	}{
		{"same name", "example/repo", false},
		{"case differs", "Example/Repo", false},
		{"transferred", "newowner/repo", true},
		{"renamed", "example/renamed", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeGitHub{override: func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path == "/repos/example/repo" {
					fmt.Fprintf(w, `{"full_name":%q,"html_url":"u","private":false,"pushed_at":"2026-01-01T00:00:00Z"}`, tc.fullName)
					return true
				}
				return false
			}}
			c, _, _ := newClient(t, f, "")
			_, err := c.FetchRepo(context.Background(), "example/repo")
			if tc.wantErr != (err != nil) {
				t.Fatalf("error %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "renamed or transferred") {
				t.Errorf("error %v", err)
			}
		})
	}
}

func TestFetchReposRequestsOnlyTheNamedRepositories(t *testing.T) {
	t.Parallel()
	f := &fakeGitHub{}
	c, _, _ := newClient(t, f, "")
	got, err := c.FetchRepos(context.Background(), []string{"example/repo", "example/other", "example/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("%d repos", len(got))
	}
	for _, p := range f.requested() {
		if !strings.HasPrefix(p, "/repos/example/repo") && !strings.HasPrefix(p, "/repos/example/other") {
			t.Errorf("unexpected request %s", p)
		}
		if strings.Contains(p, "/user") || strings.Contains(p, "/orgs") {
			t.Errorf("listing request %s", p)
		}
	}
	// Three calls for each repository, none repeated.
	if n := len(f.requested()); n != 6 {
		t.Errorf("%d requests, want 6: %v", n, f.requested())
	}
}

func TestFetchReposFailsWholeOnOneFailure(t *testing.T) {
	t.Parallel()
	c, _, _ := newClient(t, &fakeGitHub{}, "")
	got, err := c.FetchRepos(context.Background(), []string{"example/repo", "example/missing"})
	if err == nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
	if !strings.Contains(err.Error(), "example/missing") {
		t.Errorf("error does not name the repository: %v", err)
	}
}

func TestTokenIsSentOnlyWhenSet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ token, want string }{{"", ""}, {fakeToken, "Bearer " + fakeToken}} {
		f := &fakeGitHub{}
		c, _, _ := newClient(t, f, tc.token)
		if _, err := c.FetchRepo(context.Background(), "example/other"); err != nil {
			t.Fatal(err)
		}
		for _, a := range f.auths {
			if a != tc.want {
				t.Errorf("Authorization %q, want %q", a, tc.want)
			}
		}
	}
}
