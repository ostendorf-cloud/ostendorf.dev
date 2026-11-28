// Package config loads and validates this website's own environment once,
// in main, aggregating every problem into one error.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/envconf"
	"github.com/ostendorf-cloud/govue/logging"
)

// DefaultSiteURL is this site's own public origin, used when SITE_URL is
// unset: the base every robots.txt and sitemap.xml URL is built from.
const DefaultSiteURL = "https://ostendorf.dev"

const (
	// DefaultCVDCacheTTL is how long a copy of the cv document counts as
	// fresh when CVD_CACHE_TTL is unset.
	DefaultCVDCacheTTL = 5 * time.Minute
	// DefaultCVDTimeout bounds one call to cvd when CVD_TIMEOUT is unset.
	DefaultCVDTimeout = 5 * time.Second

	minCVDCacheTTL = time.Second
	maxCVDCacheTTL = 24 * time.Hour
	minCVDTimeout  = time.Second
	maxCVDTimeout  = 30 * time.Second
)

// CVD is how the site reads the cv document from cvd, the cv service.
type CVD struct {
	// URL is the base URL of cvd, without a path. Nil means cvd is not
	// configured: the site then starts and serves the placeholder on the
	// data pages.
	URL *url.URL
	// CacheTTL is the age after which a request triggers a refresh.
	CacheTTL time.Duration
	// Timeout bounds one call to cvd.
	Timeout time.Duration
}

// Config is every setting this binary reads from the environment. There is
// no secret-shaped field yet: this site serves a static SPA bundle and the
// operational endpoints, with no session, no form and no credential of its
// own.
type Config struct {
	// Addr is the public HTTP listener's address.
	Addr string
	// AdminAddr is govue's admin listener's address: /healthz, /readyz and
	// /metrics, reached only from inside the cluster, never through the
	// chart's own Ingress.
	AdminAddr string
	// SiteURL is the site's own public origin, the base every sitemap
	// and robots.txt URL is built from.
	SiteURL  *url.URL
	LogLevel slog.Level
	// TrustedProxies are the proxies whose X-Forwarded-For header is
	// believed (TRUSTED_PROXIES, comma separated CIDRs). Without them a
	// deployment behind a proxy sees only the proxy's address, and page
	// view counting then counts nothing.
	TrustedProxies []netip.Prefix
	// Umami is the connection to Umami. Its zero value counts nothing: both
	// UMAMI_URL and UMAMI_WEBSITE_ID must be set to turn counting on, and
	// exactly one of them is an error. Neither is a secret.
	Umami analytics.Config
	// CVD is the connection to cvd, from which the data pages read. None of
	// its fields is a secret: the site holds no token for cvd.
	CVD CVD
}

// Load reads every setting through envconf, so a misconfigured deployment
// reports every mistake on the first start instead of one per restart.
func Load(getenv func(string) string) (Config, error) {
	l := envconf.New(getenv)

	cfg := Config{
		Addr:      l.String("ADDR", ":8080"),
		AdminAddr: l.String("ADMIN_ADDR", ":8081"),
		SiteURL:   l.URL("SITE_URL", DefaultSiteURL),
	}

	var problems []error
	level, err := logging.Level(l.String("LOG_LEVEL", "info"))
	if err != nil {
		problems = append(problems, err)
	}
	cfg.LogLevel = level

	proxies, err := clientip.ParseCIDRs(l.String("TRUSTED_PROXIES", ""))
	if err != nil {
		problems = append(problems, err)
	}
	cfg.TrustedProxies = proxies

	umami, err := analytics.LoadConfig(getenv)
	if err != nil {
		problems = append(problems, err)
	}
	cfg.Umami = umami

	cfg.CVD = CVD{
		CacheTTL: l.Duration("CVD_CACHE_TTL", DefaultCVDCacheTTL, minCVDCacheTTL, maxCVDCacheTTL),
		Timeout:  l.Duration("CVD_TIMEOUT", DefaultCVDTimeout, minCVDTimeout, maxCVDTimeout),
	}
	// CVD_URL is optional: envconf's URL reads an empty value as missing, so
	// only a set value goes through it.
	if l.String("CVD_URL", "") != "" {
		cfg.CVD.URL = l.URL("CVD_URL", "")
	}

	if err := errors.Join(append(problems, l.Err())...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
