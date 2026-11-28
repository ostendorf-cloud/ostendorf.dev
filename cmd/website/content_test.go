package main

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

func TestRunAppStartsWithoutACVDAddressAndSaysSo(t *testing.T) {
	t.Parallel()

	_, logs := runVisit(t, testCfg(), "192.0.2.10:4711", nil)

	if got := strings.Count(logs, "cvd is not configured"); got != 1 {
		t.Errorf("log lines about a missing cvd address = %d, want 1\n%s", got, logs)
	}
}

func TestRunAppStartsWhenCVDDoesNotAnswer(t *testing.T) {
	t.Parallel()

	// The address is reserved for documentation, nothing listens there. The
	// start must not read from it: cvd is read on a request, never at startup.
	cfg := testCfg()
	cfg.CVD = config.CVD{
		URL:      &url.URL{Scheme: "http", Host: "cvd.invalid:8080"},
		CacheTTL: config.DefaultCVDCacheTTL,
		Timeout:  time.Second,
	}

	_, logs := runVisit(t, cfg, "192.0.2.10:4711", nil)

	if strings.Contains(logs, "cvd is not configured") {
		t.Errorf("log says cvd is not configured although an address is set\n%s", logs)
	}
	if strings.Contains(logs, "level=ERROR") || strings.Contains(logs, "level=WARN") {
		t.Errorf("start logged a problem for a cvd that was never asked\n%s", logs)
	}
}
