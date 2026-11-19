package webapp

import (
	"io/fs"
	"net/http"
	"net/url"

	"github.com/ostendorf-cloud/govue/analytics"
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
// the two, or nothing is counted.
func NewHandler(cfg Config, dist, prerendered fs.FS) (http.Handler, error) {
	mux, err := newMux(cfg, dist, prerendered)
	if err != nil {
		return nil, err
	}
	if cfg.Views == nil {
		return mux, nil
	}
	return cfg.Views.Middleware(mux), nil
}

// newMux builds the mux without the page view middleware.
func newMux(cfg Config, dist, prerendered fs.FS) (*http.ServeMux, error) {
	pages, err := newPageServer(dist, prerendered)
	if err != nil {
		return nil, err
	}
	h := &handler{siteURL: cfg.SiteURL}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", pages.assetHandler())
	mux.Handle("GET /robots.txt", h.robotsHandler())
	mux.Handle("GET /sitemap.xml", h.sitemapHandler())
	registerRoutes(mux, pages)
	mux.Handle("/", pages.pageHandler())
	return mux, nil
}
