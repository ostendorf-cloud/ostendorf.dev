package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// env builds a getenv function over a fixed map, the shape Load expects. A
// key absent from the map behaves exactly like an unset environment
// variable: envconf's own lookup returns "".
func env(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func TestLoadAppliesDefaultsWhenEverythingIsUnset(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.AdminAddr != ":8081" {
		t.Errorf("AdminAddr = %q, want %q", cfg.AdminAddr, ":8081")
	}
	if cfg.SiteURL.String() != DefaultSiteURL {
		t.Errorf("SiteURL = %q, want %q", cfg.SiteURL.String(), DefaultSiteURL)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoadOverridesEachVariable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		env   map[string]string
		check func(t *testing.T, cfg Config)
	}{
		{
			name: "ADDR",
			env:  map[string]string{"ADDR": ":9090"},
			check: func(t *testing.T, cfg Config) {
				if cfg.Addr != ":9090" {
					t.Errorf("Addr = %q, want %q", cfg.Addr, ":9090")
				}
			},
		},
		{
			name: "ADMIN_ADDR",
			env:  map[string]string{"ADMIN_ADDR": ":9091"},
			check: func(t *testing.T, cfg Config) {
				if cfg.AdminAddr != ":9091" {
					t.Errorf("AdminAddr = %q, want %q", cfg.AdminAddr, ":9091")
				}
			},
		},
		{
			name: "SITE_URL",
			env:  map[string]string{"SITE_URL": "https://staging.ostendorf.dev"},
			check: func(t *testing.T, cfg Config) {
				if cfg.SiteURL.String() != "https://staging.ostendorf.dev" {
					t.Errorf("SiteURL = %q, want %q", cfg.SiteURL.String(), "https://staging.ostendorf.dev")
				}
			},
		},
		{
			name: "LOG_LEVEL",
			env:  map[string]string{"LOG_LEVEL": "debug"},
			check: func(t *testing.T, cfg Config) {
				if cfg.LogLevel != slog.LevelDebug {
					t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(env(tc.env))
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			tc.check(t, cfg)
		})
	}
}

func TestLoadRejectsAnInvalidLogLevel(t *testing.T) {
	t.Parallel()

	_, err := Load(env(map[string]string{"LOG_LEVEL": "verbose"}))
	if err == nil {
		t.Fatal("Load() error = nil, want an error for an invalid LOG_LEVEL")
	}
	if !strings.Contains(err.Error(), "config:") {
		t.Errorf("Load() error = %q, want it wrapped with \"config:\"", err.Error())
	}
	if !strings.Contains(err.Error(), "verbose") {
		t.Errorf("Load() error = %q, want it to name the invalid value", err.Error())
	}
}

const (
	testUmamiURL = "https://umami.example.test"
	testUmamiID  = "0a1b2c3d-0000-4000-8000-000000000001"
)

func TestLoadCountsNothingWithoutUmamiAndRejectsAHalfSetPair(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Umami.Enabled() {
		t.Error("Umami is enabled with no UMAMI_URL and no UMAMI_WEBSITE_ID")
	}

	cfg, err = Load(env(map[string]string{"UMAMI_URL": testUmamiURL, "UMAMI_WEBSITE_ID": testUmamiID}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if !cfg.Umami.Enabled() || cfg.Umami.URL != testUmamiURL || cfg.Umami.WebsiteID != testUmamiID {
		t.Errorf("Umami = %+v, want both values carried", cfg.Umami)
	}

	for name, values := range map[string]map[string]string{
		"url only": {"UMAMI_URL": testUmamiURL},
		"id only":  {"UMAMI_WEBSITE_ID": testUmamiID},
	} {
		_, err := Load(env(values))
		if err == nil || !strings.Contains(err.Error(), "UMAMI_URL and UMAMI_WEBSITE_ID") {
			t.Errorf("%s: Load() error = %v, want the half set pair named", name, err)
		}
	}
}

func TestLoadReadsTrustedProxiesAndRejectsAMalformedOne(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{"TRUSTED_PROXIES": "10.0.0.0/8, 192.168.1.10"}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Errorf("TrustedProxies = %v, want two entries", cfg.TrustedProxies)
	}
	if _, err := Load(env(map[string]string{"TRUSTED_PROXIES": "nonsense"})); err == nil {
		t.Error("Load() error = nil, want an error for a malformed TRUSTED_PROXIES")
	}
}

func TestLoadRejectsAnInvalidSiteURL(t *testing.T) {
	t.Parallel()

	_, err := Load(env(map[string]string{"SITE_URL": "not-a-url"}))
	if err == nil {
		t.Fatal("Load() error = nil, want an error for an invalid SITE_URL")
	}
	if !strings.Contains(err.Error(), "SITE_URL") {
		t.Errorf("Load() error = %q, want it to name SITE_URL", err.Error())
	}
}

func TestLoadReadsTheCVDSettings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		env         map[string]string
		wantURL     string
		wantTTL     time.Duration
		wantTimeout time.Duration
	}{
		{
			name:        "nothing set",
			wantTTL:     DefaultCVDCacheTTL,
			wantTimeout: DefaultCVDTimeout,
		},
		{
			name:        "every value set",
			env:         map[string]string{"CVD_URL": "https://cv.example.test/", "CVD_CACHE_TTL": "30s", "CVD_TIMEOUT": "2s"},
			wantURL:     "https://cv.example.test",
			wantTTL:     30 * time.Second,
			wantTimeout: 2 * time.Second,
		},
		{
			name:        "a plain http name inside the cluster",
			env:         map[string]string{"CVD_URL": "http://cvd.cv.svc:8080"},
			wantURL:     "http://cvd.cv.svc:8080",
			wantTTL:     DefaultCVDCacheTTL,
			wantTimeout: DefaultCVDTimeout,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(env(tc.env))
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			gotURL := ""
			if cfg.CVD.URL != nil {
				gotURL = cfg.CVD.URL.String()
			}
			if gotURL != tc.wantURL {
				t.Errorf("CVD.URL = %q, want %q", gotURL, tc.wantURL)
			}
			if cfg.CVD.CacheTTL != tc.wantTTL {
				t.Errorf("CVD.CacheTTL = %v, want %v", cfg.CVD.CacheTTL, tc.wantTTL)
			}
			if cfg.CVD.Timeout != tc.wantTimeout {
				t.Errorf("CVD.Timeout = %v, want %v", cfg.CVD.Timeout, tc.wantTimeout)
			}
		})
	}
}

func TestLoadRejectsAnInvalidCVDSetting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "url without a scheme", env: map[string]string{"CVD_URL": "cv.example.test"}, want: "CVD_URL"},
		{name: "url with credentials", env: map[string]string{"CVD_URL": "https://user:pass@cv.example.test"}, want: "CVD_URL"},
		{name: "cache lifetime that is no duration", env: map[string]string{"CVD_CACHE_TTL": "soon"}, want: "CVD_CACHE_TTL"},
		{name: "cache lifetime below the minimum", env: map[string]string{"CVD_CACHE_TTL": "0s"}, want: "CVD_CACHE_TTL"},
		{name: "timeout above the maximum", env: map[string]string{"CVD_TIMEOUT": "2m"}, want: "CVD_TIMEOUT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Load(env(tc.env))
			if err == nil {
				t.Fatal("Load() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load() error = %q, want it to name %s", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadReportsEveryCVDProblemTogether(t *testing.T) {
	t.Parallel()

	_, err := Load(env(map[string]string{"CVD_URL": "nonsense", "CVD_CACHE_TTL": "soon", "CVD_TIMEOUT": "2m"}))
	if err == nil {
		t.Fatal("Load() error = nil, want an error")
	}
	for _, key := range []string{"CVD_URL", "CVD_CACHE_TTL", "CVD_TIMEOUT"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Load() error = %q, want it to name %s", err.Error(), key)
		}
	}
}
