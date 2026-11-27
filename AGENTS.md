# ostendorf.dev

The developer site of Simon Ostendorf: projects, skills and contact.

The global baseline already applies here and is not repeated below. This succeeds
old_ostendorf.dev, a plain HTML and CSS link page with no JavaScript and no third party
request; this site keeps that no-third-party-request property while becoming a govue
application.

## Layout

- `cmd/website` wires the service: config, construction and shutdown. `embed.go` embeds `dist/`
  (the built SPA, written by `task web:build`) and `prerendered/` (the static HTML per route,
  written by `task prerender`); both sit under `cmd/` rather than `internal/` because `go:embed`
  resolves relative to the file holding the directive. `main.go` builds `internal/webapp`'s
  handler on the public listener and serves govue's health, readiness and metrics on a second,
  administrative listener (`httpserver.RunAdmin`). `routes_test.go` and `thirdparty_test.go` read
  the real, generated `dist/` and `prerendered/` output rather than the source that produced it;
  see their own two bullets below.
- `internal/config` loads and validates the website's own environment once, aggregated into
  one error. `SiteURL` (`SITE_URL`, default `https://ostendorf.dev`) is the base
  `internal/webapp`'s `robots.txt` and `sitemap.xml` build every URL from. There is no
  secret-shaped variable yet: no session, no OIDC, no CSRF and no form exist in this milestone.
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
- `internal/depsync` checks that `go.mod` and `web/package.json` pin the same govue version, and
  that `web/pnpm-lock.yaml` still resolves govue's Vue package to its `web/` subdirectory; see
  "govue's two halves move together" below.
- `internal/routes` is the website's one route table: `routes.json`, checked in and embedded
  through `//go:embed`, lists every page as a name, a slug and the prerendered file that answers
  it. `internal/webapp`, `web/scripts/prerender.ts`, and the frontend's own `web/src/router/index.ts`
  and `web/src/lib/seo.ts`, through the shared `web/src/lib/routeTable.ts`, each read this one
  file to build their own list instead of restating it, so a page added to one side and not the
  other can no longer happen. `routes.go`'s `Parse` rejects a table with a missing name or file,
  or a repeated name, slug or file, before any of those five consumers ever sees it;
  `routeTable.ts`'s own `validateRouteTable` enforces the exact same three rules for the frontend,
  which reads `routes.json` through Vite's JSON import rather than through Go, so it has no Go
  validation to inherit. `internal/routes/testdata/{valid,invalid}` holds the fixtures both
  `routes_test.go` and `routeTable.test.ts` read, so a fixture either side adds is checked by both
  validators without touching the other test file.
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
- `cmd/website`'s `TestBuiltOutputMakesNoThirdPartyRequest` is the test over the built output that
  proves this site keeps old_ostendorf.dev's no-third-party-request property (see the top of this
  file): it reads the real `dist/` and `prerendered/` bytes for every element whose `src` or
  `href` the browser fetches on its own (`script`, `link` with a fetching `rel`, `img`, `source`,
  `iframe`, `video`, `audio`, `embed`, `object`) and every CSS `@import`/`url()`, and fails on an
  absolute `http(s)://` reference. A plain `<a href>` is deliberately excluded: the home page's
  own links to GitHub, LinkedIn and a `mailto:` address are the visitor's own choice to follow,
  not a request the page causes by itself.
- `web/` is the Vue SPA, consuming govue's Vue package as a pnpm git dependency.
  `web/src/router/index.ts` builds one `vue-router` route per `internal/routes` entry;
  `web/src/lib/seo.ts` resolves each route's title, meta description and canonical URL the same
  way. `web/src/App.vue` is the page shell: `GovueSkipLink`, `AppHeader` (brand, `GovueNav`,
  `ThemeToggle`), `<router-view>`, `AppFooter`. `ThemeToggle.vue` cycles govue's `useTheme` mode
  (light, dark, system) rather than hand rolling a theme switch. `web/src/views/HomeView.vue` is
  the one real page this milestone ships: the predecessor's own heading and profile links (see
  "Linking to the sibling sites" below), nothing more; `NotFoundView.vue` answers every path the
  route table does not know. `web/scripts/prerender.ts` is the static site generation step
  `task prerender` runs, reading the same route table through `routeTable.ts`.
  `web/src/theme/ostendorf-dev.css` fills govue's design token contract. It is generated by
  govue's theme generator (`web/scripts/generate-theme.ts` in the govue module) from the two seed
  colours in `web/src/theme/seeds.json`, and checked against WCAG 2.2 AA by
  `web/src/theme/contrast.test.ts`. To change the palette, edit the seeds and regenerate; never
  edit the CSS by hand.
- `charts/ostendorf.dev/` is the Helm chart. Its `Chart.yaml` `name` and every rendered object
  name use `ostendorf-dev` instead of `ostendorf.dev`: a literal dot is invalid in a `Service`'s
  `metadata.name` (RFC 1035 label), which a chart named after the repository directly would
  otherwise produce by default.
- `Dockerfile` and `docker-compose.yaml` build and run the one binary locally.
- `scripts/smoke-test-image.sh` starts a built image and asserts it answers health, readiness and
  metrics on the admin listener and a request on the public one; the `docker-build` job in
  `.github/workflows/checks.yaml` runs it right after building the image.

## Linking to the sibling sites

ostendorf.dev's own navigation and footer carry no link to simon-ostendorf.de or ostendorf.tech.
`docs/concepts/ostendorf-dev.md` already settles the boundary between the three sites:
simon-ostendorf.de is the neutral hub that carries the identity-level summary and the outbound
links to the other two, and ostendorf.tech and ostendorf.dev each carry the deep, domain specific
content only. A footer or nav link here to either sibling would restate what the hub already
does, the exact thing that boundary says this site should not do. `AppFooter.vue`'s own comment
points back here.

## A build needs the frontend first

`cmd/website` cannot compile against an empty `dist/` or `prerendered/`, because `go:embed` fails
when its pattern matches no file at all. Both are gitignored, generated by `task web:build` and
`task prerender` respectively.

`task build` and `task test:go` both declare that dependency, so running either one through Task
always populates the frontend first. A bare `go build ./cmd/website` or `go test ./cmd/website/...`
outside Task fails on a fresh checkout for that reason; run `task build` or `task test:go` instead,
or `task web:build && task prerender` once by hand before reaching for the Go toolchain directly.

## govue's two halves move together

govue releases a Go module and a Vue package under one tag. `go.mod`'s
`require github.com/ostendorf-cloud/govue` and `web/package.json`'s `@ostendorf-cloud/govue`
pnpm git dependency must always name the same tag: a mismatch means the repository takes half of
a govue release and not the other half, silently. `task check:govue-version`
(`internal/depsync`) enforces this: it asks `go list -m` for the version go.mod actually
resolves to, the same resolution `go build` itself uses, and pulls the tag out of
package.json's `ostendorf-cloud/govue#<tag>&path:/web` specifier, then fails naming both
versions and both files when they differ. `web/pnpm-lock.yaml`'s own specifier field already
has to match package.json's for `pnpm install --frozen-lockfile` to pass, and its resolved entry
pins a commit hash, not a tag, so it cannot be compared to a version directly either. That match
is not enough on its own, though: pnpm's own git resolver has written a resolution object for
this dependency with no `path` field at all, even from a fully fresh resolve against a cold
store, which makes `pnpm install --frozen-lockfile` install the whole govue repository instead
of its `web/` subdirectory while every specifier still agrees (pnpm/pnpm#12304, fixed by
pnpm/pnpm#12344, not backported to the pnpm 10 release line as far as this repository could
establish). `check:govue-version` therefore also reads `web/pnpm-lock.yaml` directly and fails
when that dependency's resolution carries no `path: /web`.

Renovate raises both halves as one pull request. `renovate.json` groups the Go module and the Vue
package as `govue`, and a regex custom manager reads the tag from `web/package.json`, because
Renovate's npm manager skips the `&path:/web` specifier. The shared preset `govue-lockfile`
regenerates `web/pnpm-lock.yaml`. The group's `constraints.pnpm` equals the pin in `mise.toml`:
move both together, and a `pnpm` group moves the two pins in one pull request.
`internal/depsync/renovate_test.go` fails when any of this goes missing.

## Verify before committing

```sh
task check
```

`task --list` shows the individual targets, including the small ones `task check` aggregates.

## Scope of this milestone

The page shell, the one route table and prerendering are implemented: the navigation, the
footer, the theme switch, the home page's own heading and the predecessor's profile links,
`robots.txt`/`sitemap.xml`, the site's own 404 page, and the no-third-party-request test over the
built output. None of the site's other content is implemented yet. Left out, tracked as later
milestones, each adding its own route to `internal/routes/routes.json` and its own nav item:

- Projects, read at runtime from `cvd`, the `cv` repository's service, with their hand written
  descriptions and the GitHub languages and latest release `cvd` holds. This waits on two
  changes in `cv`: GitHub enrichment in `cvd` and a public client package. ADR 0001 records
  the cache, retry and placeholder policy, and the concept doc the configuration.
- Skills and tools, read from the same `cvd` document.
- A contact form, ported later from `ostendorf.tech`: proof of work in the browser, SMTP
  delivery, no third party.
- A uses page: setup, tools, homelab.

Each of these adds a view, a route table entry and a nav item onto the shell this milestone
built; the contact form also needs its own configuration (`internal/config` carries none of its
settings yet) and its own API routes on `internal/webapp`'s handler, plus the session manager and
CSRF protector the concept doc describes, none of which exist yet.

## Deviations

- `LICENSE` is proprietary, all rights reserved, not the baseline's Apache-2.0: this is one
  person's own site, and reusing it would misidentify the source of a service. No per-file
  licence header and no reusable license-header workflow for the same reason; the single
  `LICENSE` carries it. `NOTICE` records what `LICENSE` does not cover.
- The Helm chart lives at `charts/ostendorf.dev/`, matching the repository's own name, but its
  `Chart.yaml` `name` and every rendered object name use `ostendorf-dev` instead: a literal dot
  is invalid in a `Service`'s `metadata.name` (RFC 1035 label), which a chart named
  `ostendorf.dev` would otherwise produce by default.
- `helm-unittest` is pinned in `mise.toml` through mise's generic `github:` backend with a
  `postinstall` hook, not a plain registry entry: it is not in the mise registry, and
  `helm plugin install <url>` is forbidden by the baseline regardless of Helm version.
