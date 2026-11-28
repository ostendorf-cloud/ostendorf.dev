package webapp

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/ostendorf-cloud/cv/client"
)

// retryAfterSeconds is the Retry-After value of the placeholder. It matches
// the gap the content store waits after a failed read: a retry sooner than
// that would only meet the same placeholder.
const retryAfterSeconds = "30"

// DocumentSource returns the cv document of one language. *content.Store is
// the implementation. A nil result with an error means no copy exists yet.
type DocumentSource interface {
	Get(ctx context.Context, lang string) (*client.Document, error)
}

// dataHandler answers a page that renders the cv document of lang. When the
// source has no copy, it answers the 503 placeholder instead and never calls
// render. A nil source behaves like one that has no copy: that is how a site
// without CVD_URL serves its data pages.
func (p *pageServer) dataHandler(lang string, render func(http.ResponseWriter, *http.Request, *client.Document)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.content == nil {
			p.serveUnavailable(w)
			return
		}
		doc, err := p.content.Get(r.Context(), lang)
		if err != nil || doc == nil {
			p.serveUnavailable(w)
			return
		}
		render(w, r, doc)
	})
}

// serveUnavailable answers 503 with the site's own prerendered placeholder
// and a Retry-After header, falling back to a plain text line when the file
// cannot be read.
func (p *pageServer) serveUnavailable(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Retry-After", retryAfterSeconds)
	h.Set("Cache-Control", "no-store")

	data, err := fs.ReadFile(p.prerendered, "unavailable.html")
	if err != nil {
		http.Error(w, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}
	h.Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write(data)
}
