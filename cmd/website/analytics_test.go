package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/httpserver"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

const (
	testWebsiteID = "0a1b2c3d-0000-4000-8000-000000000001"
	testUmamiURL  = "https://umami.example.test"
	visitorUA     = "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"
)

// fakeUmami is the transport standing in for Umami: it records the bodies.
type fakeUmami struct {
	mu    sync.Mutex
	posts []string
}

func (f *fakeUmami) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	f.mu.Lock()
	f.posts = append(f.posts, string(body))
	f.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func (f *fakeUmami) bodies() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.posts...)
}

func countingCfg(proxies ...string) config.Config {
	cfg := testCfg()
	cfg.Umami = analytics.Config{URL: testUmamiURL, WebsiteID: testWebsiteID}
	for _, p := range proxies {
		cfg.TrustedProxies = append(cfg.TrustedProxies, netip.MustParsePrefix(p))
	}
	return cfg
}

// runVisit goes through runApp with a stand in for the listeners: serve
// sends one request through the real public handler, as a visitor would, and
// returns. Whatever reaches Umami therefore got there through the real
// wiring, and only because runApp closed the client after serve returned.
func runVisit(t *testing.T, cfg config.Config, peer string, header http.Header) ([]string, string) {
	t.Helper()
	fake := &fakeUmami{}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	serve := func(ctx context.Context, public httpserver.Config, _ httpserver.AdminConfig) error {
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/?token=s3cret", nil)
		req.RemoteAddr = peer
		req.Header.Set("User-Agent", visitorUA)
		for k, v := range header {
			req.Header[k] = v
		}
		rec := httptest.NewRecorder()
		public.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET / = %d, want %d", rec.Code, http.StatusOK)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if err := runApp(ctx, cfg, logger, serve, &http.Client{Transport: fake}); err != nil {
		t.Fatalf("runApp() error = %v", err)
	}
	return fake.bodies(), logs.String()
}

type sentView struct {
	URL string `json:"url"`
	IP  string `json:"ip"`
}

func decode(t *testing.T, body string) sentView {
	t.Helper()
	var ev struct {
		Payload sentView `json:"payload"`
	}
	if err := json.Unmarshal([]byte(body), &ev); err != nil {
		t.Fatalf("Umami body %q is not JSON: %v", body, err)
	}
	return ev.Payload
}

func TestRunAppSendsTheMaskedAddressOfTheClientBehindATrustedProxy(t *testing.T) {
	bodies, _ := runVisit(t, countingCfg("10.0.0.0/8"), "10.0.0.2:4000",
		http.Header{"X-Forwarded-For": {"198.51.100.77"}})

	if len(bodies) != 1 {
		t.Fatalf("Umami received %d posts, want 1 (runApp must send the queue on shutdown): %v", len(bodies), bodies)
	}
	got := decode(t, bodies[0])
	if got.URL != "/" {
		t.Errorf("url = %q, want the route pattern /", got.URL)
	}
	if got.IP != "198.51.100.0" {
		t.Errorf("ip = %q, want the /24 of the forwarded client, 198.51.100.0", got.IP)
	}
	if strings.Contains(bodies[0], "198.51.100.77") || strings.Contains(bodies[0], "s3cret") {
		t.Errorf("Umami body %s carries the whole address or the query", bodies[0])
	}
}

func TestRunAppIgnoresForwardedForFromAnUntrustedPeer(t *testing.T) {
	bodies, _ := runVisit(t, countingCfg("10.0.0.0/8"), "203.0.113.9:4000",
		http.Header{"X-Forwarded-For": {"198.51.100.77"}})

	if len(bodies) != 1 {
		t.Fatalf("Umami received %d posts, want 1: %v", len(bodies), bodies)
	}
	if ip := decode(t, bodies[0]).IP; ip != "203.0.113.0" {
		t.Errorf("ip = %q, want the peer's /24, 203.0.113.0", ip)
	}
}

func TestRunAppCountsNothingBehindAProxyWhenNoProxyIsTrusted(t *testing.T) {
	bodies, logs := runVisit(t, countingCfg(), "10.0.0.2:4000",
		http.Header{"X-Forwarded-For": {"198.51.100.77"}})

	if len(bodies) != 0 {
		t.Errorf("Umami received %v, want nothing: the peer is a private address", bodies)
	}
	if !strings.Contains(logs, "level=WARN") || !strings.Contains(logs, "TRUSTED_PROXIES") {
		t.Errorf("startup log %q has no warning about the empty TRUSTED_PROXIES", logs)
	}
}

func TestRunAppCountsNothingAndWarnsOfNothingWhenCountingIsOff(t *testing.T) {
	bodies, logs := runVisit(t, testCfg(), "203.0.113.9:4000", nil)

	if len(bodies) != 0 {
		t.Errorf("Umami received %v with counting off", bodies)
	}
	if !strings.Contains(logs, "page view counting") || !strings.Contains(logs, "enabled=false") {
		t.Errorf("startup log %q does not say counting is off", logs)
	}
	if strings.Contains(logs, "level=WARN") {
		t.Errorf("startup log %q warns with counting off", logs)
	}
}

func TestLogViewsStatusWarnsOnlyForCountingWithoutTrustedProxies(t *testing.T) {
	t.Parallel()

	on, err := analytics.New(analytics.Config{URL: testUmamiURL, WebsiteID: testWebsiteID},
		analytics.Options{Hostname: "ostendorf.dev", Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("analytics.New() error = %v", err)
	}
	t.Cleanup(func() { _ = on.Close(context.Background()) })

	cases := map[string]struct {
		proxies []netip.Prefix
		warns   bool
	}{
		"counting, no proxy": {warns: true},
		"counting, proxy":    {proxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			cfg := config.Config{SiteURL: &url.URL{Host: "ostendorf.dev"}, TrustedProxies: tc.proxies}
			logViewsStatus(slog.New(slog.NewTextHandler(&logs, nil)), cfg, on)
			if got := strings.Contains(logs.String(), "level=WARN"); got != tc.warns {
				t.Errorf("warned = %v, want %v, log %q", got, tc.warns, logs.String())
			}
		})
	}
}
