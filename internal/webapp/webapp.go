package webapp

import (
	"io/fs"
	"net/http"
	"net/url"
)

// Config is what NewHandler needs beyond the two filesystems it serves
// from.
type Config struct {
	// SiteURL is the site's own public origin, the base every
	// robots.txt and sitemap.xml URL is built from.
	SiteURL *url.URL
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
func NewHandler(cfg Config, dist, prerendered fs.FS) (http.Handler, error) {
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
