// Package webapp builds the website's one HTTP handler: the built SPA's
// assets, the prerendered HTML per route, robots.txt and sitemap.xml, all
// read from internal/routes' one route table. See AGENTS.md for how each
// of this table's five consumers reads it.
package webapp

import (
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/routes"
)

// routeFiles maps a route slug to the prerendered file that answers it.
// The empty slug is the home page. Every value is relative to
// cmd/website/prerendered.
//
// Built from internal/routes' embedded route table, the single list
// web/scripts/prerender.ts also reads to decide what it writes: adding a
// page means editing that one file, routes.json, not this map. A
// malformed routes.json is a build bug, caught by internal/routes' own
// tests before it ever reaches this package; see mustRouteFiles.
var routeFiles = mustRouteFiles()

// mustRouteFiles loads the embedded route table and indexes it by slug.
// A load failure here can only mean routes.json itself is malformed: it
// is checked in, not a runtime input, so a panic here is the right
// response rather than threading an error through every caller of this
// package level table.
func mustRouteFiles() map[string]string {
	table, err := routes.Load()
	if err != nil {
		panic("webapp: " + err.Error())
	}
	files := make(map[string]string, len(table))
	for _, r := range table {
		files[r.Slug] = r.File
	}
	return files
}

// routeSlugs lists routeFiles' keys in a deterministic, sorted order, so
// registering one http.ServeMux pattern per slug never depends on Go's
// unspecified map iteration order.
func routeSlugs() []string {
	slugs := make([]string, 0, len(routeFiles))
	for slug := range routeFiles {
		slugs = append(slugs, slug)
	}
	slices.Sort(slugs)
	return slugs
}

// routePattern builds the http.ServeMux pattern slug registers under,
// with {$} anchoring to the exact end of the path so the pattern never
// matches a child path or a request missing its trailing slash.
func routePattern(slug string) string {
	if slug == "" {
		return "GET /{$}"
	}
	return "GET /" + slug + "/{$}"
}

// registerRoutes registers one http.ServeMux pattern per route slug on
// mux. It is the one place that loops over routeSlugs, so a test
// exercising the real registration never has to hand-duplicate a route
// pattern string.
func registerRoutes(mux *http.ServeMux, p *pageServer) {
	for _, slug := range routeSlugs() {
		mux.Handle(routePattern(slug), p.routeHandler(slug))
	}
}

// pageServer answers every document and static asset request out of the
// built SPA and the prerendered HTML, both read only filesystems.
type pageServer struct {
	assets      fs.FS
	prerendered fs.FS
	// routes is routeFiles, carried on the instance rather than read
	// directly off the package level var so pageHandler's redirect gate
	// can be exercised in a test against a route set routes.json itself
	// does not have to provide.
	routes map[string]string
}

// newPageServer validates that dist carries the assets/ directory the SPA
// build writes, so a missing build is reported once at startup rather than
// as a 404 on the first request. fs.Sub alone would not catch this: it
// builds a filesystem rooted at the given directory lazily, with no check
// that the directory exists, for every fs.FS that does not implement
// fs.SubFS itself (embed.FS included), so the explicit fs.Stat below is
// what actually does the validating.
func newPageServer(dist, prerendered fs.FS) (*pageServer, error) {
	if _, err := fs.Stat(dist, "assets"); err != nil {
		return nil, fmt.Errorf("webapp: dist has no assets directory: %w", err)
	}
	assets, err := fs.Sub(dist, "assets")
	if err != nil {
		return nil, fmt.Errorf("webapp: dist has no assets directory: %w", err)
	}
	return &pageServer{assets: assets, prerendered: prerendered, routes: routeFiles}, nil
}

// assetHandler serves the SPA's built, content hashed files as immutable:
// the filename changes with the content, so a cached copy can never go
// stale.
func (p *pageServer) assetHandler() http.Handler {
	fileServer := http.FileServerFS(p.assets)
	files := http.StripPrefix("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A path ending in a slash names a directory, which http.FileServerFS
		// would answer with a 200 HTML listing: no listing is served, and
		// page view counting must never mistake one for a page.
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}

// routeHandler answers one known route slug, registered once at startup
// under routePattern(slug) so routeFiles[slug] can be captured directly:
// the pattern itself pins the slug literally, so no per-request lookup or
// validation is needed.
func (p *pageServer) routeHandler(slug string) http.Handler {
	file := routeFiles[slug]
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		p.serveFile(w, http.StatusOK, file)
	})
}

// pageHandler is the catch-all document handler: trailing slash
// canonicalisation for a known route, and the site's own 404 page for
// anything else. Serving a known route's prerendered HTML is handled by
// the per-slug patterns routeHandler registers; those intercept every
// known route that already carries its trailing slash in real serving,
// so this handler only ever reaches the 404 branch for a genuinely
// unknown path, or canonicalises a known route missing its trailing
// slash before a visitor ever sees it.
//
// isKnownRoute is the gate the reference implementation, ostendorf.tech,
// has in the equivalent place through its locale check: a path that does
// not start with a known locale answers 404 there before the trailing
// slash branch is ever reached. This site serves one locale, so there is
// no locale segment to check; isKnownRoute is what fills that place
// instead, so an unknown path is never redirected to a target that would
// only then answer 404 itself.
func (p *pageServer) pageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		if !strings.HasSuffix(path, "/") {
			if !p.isKnownRoute(path) {
				p.serveNotFound(w)
				return
			}
			target := path + "/"
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}

		p.serveNotFound(w)
	})
}

// isKnownRoute reports whether path, once canonicalised with a trailing
// slash, names a route slug in p.routes.
func (p *pageServer) isKnownRoute(path string) bool {
	_, ok := p.routes[strings.Trim(path, "/")]
	return ok
}

// serveFile writes the prerendered file at name, or falls back to the
// site's own 404 page when it is missing: a route this handler believes
// exists but whose file never got prerendered must still answer something
// rather than crash.
func (p *pageServer) serveFile(w http.ResponseWriter, status int, name string) {
	data, err := fs.ReadFile(p.prerendered, name)
	if err != nil {
		p.serveNotFound(w)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// serveNotFound answers 404 with the site's own prerendered page, falling
// back to a plain text line when even that cannot be read.
func (p *pageServer) serveNotFound(w http.ResponseWriter) {
	data, err := fs.ReadFile(p.prerendered, "404.html")
	if err != nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(data)
}
