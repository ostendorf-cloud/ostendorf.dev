package webapp

import (
	"fmt"
	"net/http"
	"strings"
)

// robotsHandler serves /robots.txt: every route is open to crawling, there
// is no route this site wants hidden, plus the sitemap's own location.
func (h *handler) robotsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", h.siteBase())
	})
}

// sitemapHandler serves /sitemap.xml: one <url> per route routeFiles
// knows. routeFiles is the exact route table pageHandler serves from, so
// this sitemap can never list a route the server does not actually
// answer, nor miss one it does.
func (h *handler) sitemapHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")

		slugs := routeSlugs(h.routes)

		base := h.siteBase()

		var buf strings.Builder
		buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
		for _, slug := range slugs {
			buf.WriteString("  <url>\n")
			fmt.Fprintf(&buf, "    <loc>%s</loc>\n", routeURL(base, slug))
			buf.WriteString("  </url>\n")
		}
		buf.WriteString("</urlset>\n")
		_, _ = w.Write([]byte(buf.String()))
	})
}

// siteBase is the configured site origin with no trailing slash, the base
// every sitemap and robots.txt URL is built from.
func (h *handler) siteBase() string {
	return strings.TrimSuffix(h.siteURL.String(), "/")
}

// routeURL is the absolute URL of slug (a routeFiles key).
func routeURL(base, slug string) string {
	if slug == "" {
		return base + "/"
	}
	return base + "/" + slug + "/"
}
