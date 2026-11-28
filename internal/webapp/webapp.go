package webapp

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/language"
	"github.com/ostendorf-cloud/govue/logging"
	"github.com/ostendorf-cloud/govue/middleware"
)

// Config is what NewHandler needs beyond the two filesystems it serves
// from.
type Config struct {
	// SiteURL is the site's own public origin, the base every
	// robots.txt and sitemap.xml URL is built from.
	SiteURL *url.URL
	// Views counts page views on the server. Nil, or a client that is not
	// enabled, counts nothing.
	Views *analytics.Client
	// Logger receives the access log, one line per request. Nil writes no
	// access log.
	Logger *slog.Logger
	// ClientIP resolves the client address the access log writes, behind
	// the trusted proxies. Nil leaves the address out of the log.
	ClientIP *clientip.Resolver
	// Content supplies the cv document to the pages that render it. Nil
	// means cvd is not configured, and those pages answer the 503
	// placeholder.
	Content DocumentSource
}

// handler carries the one piece of request-independent state the sitemap
// and robots.txt handlers close over.
type handler struct {
	siteURL *url.URL
}

// NewHandler builds the website's one HTTP handler: the built SPA's
// content hashed assets, the prerendered HTML per route (serving the
// site's own 404 page for anything else, and canonicalising a missing
// trailing slash first), and robots.txt/sitemap.xml, all derived from
// internal/routes' one route table.
//
// When cfg.Views is set, its middleware wraps the mux directly. That
// position is load bearing: it reports the route pattern the mux sets on
// the request, so no middleware that replaces the request may sit between
// the two, or nothing is counted. When cfg.Logger is set, govue's access
// log wraps the page view middleware and reads the same pattern; only the
// small middleware that puts the logger into the request context sits
// outside it, because that one replaces the request.
func NewHandler(cfg Config, dist, prerendered fs.FS) (http.Handler, error) {
	mux, err := newMux(cfg, dist, prerendered)
	if err != nil {
		return nil, err
	}
	return wrap(cfg, mux), nil
}

// wrap puts the page view counter and the access log around inner. The
// access log has no user agent and no referrer: only what AccessLog writes
// by default, plus the client address when a resolver is set.
func wrap(cfg Config, inner http.Handler) http.Handler {
	h := inner
	if cfg.Views != nil {
		h = cfg.Views.Middleware(h)
	}
	if cfg.Logger == nil {
		return h
	}
	accessLog := middleware.AccessLog{ClientIP: cfg.ClientIP}
	return withLogger(cfg.Logger, accessLog.Middleware(h))
}

// withLogger carries logger in the request context, where AccessLog reads
// it. It replaces the request, so it sits outside the access log.
func withLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(logging.NewContext(r.Context(), logger)))
	})
}

// newMux builds the mux without the page view middleware.
func newMux(cfg Config, dist, prerendered fs.FS) (*http.ServeMux, error) {
	pages, err := newPageServer(dist, prerendered)
	if err != nil {
		return nil, err
	}
	pages.content = cfg.Content
	h := &handler{siteURL: cfg.SiteURL}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", pages.assetHandler())
	mux.Handle("GET /robots.txt", h.robotsHandler())
	mux.Handle("GET /sitemap.xml", h.sitemapHandler())
	registerRoutes(mux, pages)
	// The short and German legal paths redirect to the locale pages the route
	// table serves. No terms: the site has none, so /terms and /agb stay 404.
	// Registered on the mux itself, so the page view middleware sees them,
	// and they answer 302, which it never counts.
	if err := language.HandleLegal(mux, language.LegalConfig{}, nil); err != nil {
		return nil, fmt.Errorf("webapp: %w", err)
	}
	mux.Handle("/", pages.pageHandler())
	return mux, nil
}
