package config

import (
	"log/slog"
	"strings"
	"testing"
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
