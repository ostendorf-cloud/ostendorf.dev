# internal/webapp

## Handler and routes

- `internal/webapp` builds the website's one HTTP handler from `internal/routes`' table:
  `pages.go`'s `routeFiles` indexes that table by slug, registers one `http.ServeMux` pattern per
  route (`registerRoutes`), serves the built SPA's content hashed assets as immutable, and falls
  back to the site's own prerendered 404 page for anything else, canonicalising a missing
  trailing slash first. `sitemap.go` serves `/robots.txt` and `/sitemap.xml` from that same
  `routeFiles`, so neither can list or allow a route the handler does not actually serve. That a
  page added to `routes.json` actually gets prerendered, rather than merely agreeing with Go's own
  copy of a list that no longer exists, is what `cmd/website`'s
  `TestPrerenderedOutputMatchesRouteTable` checks (`task test:routes`): it walks the real output
  `task prerender` wrote and compares it against `internal/routes`, catching both a route with no
  prerendered file and a prerendered file that answers no route.

## Analytics and access log

- Page views are counted on the server by govue's `analytics` package, adopted as is. Config
  (`UMAMI_URL`, `UMAMI_WEBSITE_ID`, `TRUSTED_PROXIES`) lives in `internal/config`, the client is
  built in `cmd/website/main.go` and closed on shutdown, and `internal/webapp.NewHandler` wraps
  the mux with its middleware directly: nothing that replaces the request may sit between the
  two, or the route pattern is lost and nothing is counted. The chain is, outermost first, a
  small logger carrier, govue's `middleware.AccessLog` (client address from a resolver built from
  the same `TRUSTED_PROXIES` configuration as the analytics one, no user agent, one line per
  request to the slog logger, route pattern only), the analytics middleware, the mux. The privacy
  text states this application log and the ingress log separately. `docs/analytics.md` states
  what is counted and logged, and is the source for the privacy text.

## Legal pages

- The legal pages are the site's own text on govue's structure. `internal/webapp` mounts
  `language.HandleLegal` (no terms: the site has none, so `/terms` and `/agb` stay 404), which
  redirects `/imprint`, `/privacy` and the German aliases to `/de/...` or `/en/...`. The four
  pages are `routes.json` entries with `"exact": true`, served at `/de/imprint` and so on without
  a trailing slash, the path govue's footer links and `govue-check-legal` expect. Their text is
  `web/src/views/ImprintView.vue` and `PrivacyView.vue`, both languages in one file, operator
  facts in `web/src/legal/operator.ts`. `task prerender` ends with `govue-check-legal`, and
  `web/scripts/prerender.ts` writes a build time `sitemap.xml` that `cmd/website`'s
  `TestPrerenderedSitemapIsTheServedSitemap` proves equal to the served one. The imprint has no
  dispute resolution section and the privacy text names no contact form, since there is none. The
  home page stays English at `/`.
