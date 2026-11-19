package webapp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/clientip"
)

const (
	proxyAddr  = "10.1.2.3:4000"
	clientAddr = "198.51.100.9"
)

func accessLogConfig(logs *bytes.Buffer) Config {
	cfg := testConfig()
	cfg.Logger = slog.New(slog.NewJSONHandler(logs, nil))
	cfg.ClientIP = clientip.NewResolver([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	return cfg
}

func logLines(t *testing.T, logs *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("log line %q is not JSON: %v", raw, err)
		}
		lines = append(lines, line)
	}
	return lines
}

// TestAccessLogWritesOneLineWithThePatternAndTheTrustedClient goes through
// the real handler chain, with the page view counter innermost as in
// production.
func TestAccessLogWritesOneLineWithThePatternAndTheTrustedClient(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	fake := &fakeUmami{}
	cfg := accessLogConfig(&logs)
	views, err := analytics.New(
		analytics.Config{URL: testUmamiURL, WebsiteID: testWebsiteID},
		analytics.Options{
			Hostname:   "ostendorf.dev",
			Logger:     slog.New(slog.DiscardHandler),
			HTTPClient: &http.Client{Transport: fake},
			ClientIP:   cfg.ClientIP,
		})
	if err != nil {
		t.Fatalf("analytics.New() error = %v", err)
	}
	cfg.Views = views
	dist, prerendered := fakeBuild()
	h, err := NewHandler(cfg, dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/?token=s3cret", nil)
	req.RemoteAddr = proxyAddr
	req.Header.Set("X-Forwarded-For", clientAddr)
	req.Header.Set("User-Agent", visitorUA)
	req.Header.Set("Referer", "https://referrer.example.test/secret-path?x=1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", rec.Code, http.StatusOK)
	}

	lines := logLines(t, &logs)
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want exactly 1: %s", len(lines), logs.String())
	}
	line := lines[0]
	if line["client"] != clientAddr {
		t.Errorf("client = %v, want %s, the address behind the trusted proxy", line["client"], clientAddr)
	}
	if route, _ := line["route"].(string); route == "" || route == "unmatched" {
		t.Errorf("route = %v, want the mux pattern", line["route"])
	}
	if line["status"] != float64(http.StatusOK) || line["method"] != http.MethodGet {
		t.Errorf("line = %v, want method GET and status 200", line)
	}
	for _, key := range []string{"bytes", "duration_ms"} {
		if _, ok := line[key]; !ok {
			t.Errorf("line has no %q: %v", key, line)
		}
	}
	for _, key := range []string{"user_agent", "referrer", "referer", "path", "url", "query"} {
		if _, ok := line[key]; ok {
			t.Errorf("line carries %q: %v", key, line)
		}
	}
	for _, leaked := range []string{"s3cret", "token", "Mozilla", "referrer.example", "secret-path"} {
		if strings.Contains(logs.String(), leaked) {
			t.Errorf("the log carries %q: %s", leaked, logs.String())
		}
	}

	// The access log sits outside the counter, which still sees the pattern.
	settle(t, cfg.Views)
	if got := viewURLs(t, fake.bodies()); len(got) != 1 || got[0] != "/" {
		t.Errorf("reported URLs = %v, want exactly [/]", got)
	}
}

func TestAccessLogLogsThePatternNotThePathOfAnUnknownRoute(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	dist, prerendered := fakeBuild()
	h, err := NewHandler(accessLogConfig(&logs), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	visit(t, h, http.MethodGet, "/nowhere/s3cret?token=s3cret", nil)

	if len(logLines(t, &logs)) != 1 {
		t.Fatalf("want exactly one log line: %s", logs.String())
	}
	for _, leaked := range []string{"nowhere", "s3cret", "token"} {
		if strings.Contains(logs.String(), leaked) {
			t.Errorf("the log carries %q: %s", leaked, logs.String())
		}
	}
}

func TestAccessLogIgnoresForwardedForFromAnUntrustedPeer(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	dist, prerendered := fakeBuild()
	h, err := NewHandler(accessLogConfig(&logs), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	visit(t, h, http.MethodGet, "/", http.Header{"X-Forwarded-For": {"192.0.2.66"}})

	lines := logLines(t, &logs)
	if len(lines) != 1 || lines[0]["client"] != "203.0.113.7" {
		t.Fatalf("lines = %v, want the peer address 203.0.113.7", lines)
	}
}

func TestNoLoggerWritesNoAccessLog(t *testing.T) {
	t.Parallel()

	dist, prerendered := fakeBuild()
	h, err := NewHandler(testConfig(), dist, prerendered)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if rec := visit(t, h, http.MethodGet, "/", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestAPanicIsLoggedAs500AndReachesTheServer: the access log observes the
// panic and raises it again unchanged, so net/http still handles it as it
// did before the access log existed.
func TestAPanicIsLoggedAs500AndReachesTheServer(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := wrap(accessLogConfig(&logs), boom)

	func() {
		defer func() {
			if got := recover(); got != "boom" {
				t.Errorf("recovered %v, want the original panic value", got)
			}
		}()
		visit(t, h, http.MethodGet, "/", nil)
	}()

	lines := logLines(t, &logs)
	if len(lines) != 1 || lines[0]["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("lines = %v, want one line with status 500", lines)
	}
}
