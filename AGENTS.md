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
  `web/src/theme/ostendorf-dev.css` fills govue's design token contract with a neutral,
  dark-first placeholder palette, checked against WCAG 2.2 AA by
  `web/src/theme/contrast.test.ts`; see that file's own comment for why it is not the final
  brand.
- `charts/ostendorf.dev/` is the Helm chart. Its `Chart.yaml` `name` and every rendered object
  name use `ostendorf-dev` instead of `ostendorf.dev`: a literal dot is invalid in a `Service`'s
  `metadata.name` (RFC 1035 label), which a chart named after the repository directly would
  otherwise produce by default.
- `internal/content` and `cmd/content` are the build time data pipeline; see "The content data
  pipeline" below. Nothing in them runs in the deployed service.
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

## The content data pipeline

`task content:build` runs `cmd/content`. It downloads `cv.de.json` and `cv.en.json` from the cv
repository's release named in `cv-release.env` (`CV_RELEASE_TAG`), calls the GitHub REST API for
every repository the artefact's `projects` name, merges both and writes
`internal/content/generated/dataset.json`. `internal/content` embeds that directory with
`//go:embed all:generated`, and `cmd/website` loads it at start. The deployed binary holds no
token, makes no call and has no new failure mode beyond a missing dataset, which the Dockerfile
refuses to build.

- The pin is a release tag plus the sha256 of both artefacts, all in `cv-release.env`:
  `CV_RELEASE_TAG`, `CV_RELEASE_SHA256_DE`, `CV_RELEASE_SHA256_EN`. `ValidateTag` accepts a
  version tag such as `v1.0.0` and rejects `latest`, a branch, an empty value and anything that
  could change a URL path. The tag alone is not immutable: this repository does not rely on
  immutable cv releases, so a moved tag or a replaced asset would change the build silently. The
  trust anchor is therefore the two digests in this repository. `CheckDigest` refuses a mismatch,
  an empty digest next to a set tag, and a value that is not 64 hex digits (upper case is
  accepted). The message names `cv-release.env` and the tag and prints the two digests, nothing
  more. The dataset also records the digests it read (`source.cv.sha256`). Bumps are manual: no
  Renovate manager covers the file. To bump, download both assets of the new release, run
  `sha256sum cv.de.json cv.en.json` over them, and set the tag and both digests together in one
  commit. The pin is empty until the cv repository publishes its first release, so
  `task content:build`, `task build` and the CI jobs that call them fail with "no cv release is
  pinned" until a person sets it. No sample fallback exists on purpose: failing loudly is right.
  With `-artefact-dir` the build reads local files, checks no tag or digest, and records
  `source.cv.release` as `local`.
- The artefact's `schema_version` is checked first. Only version 1 is read. Any other version
  fails the build and names the version and the fix. Unknown keys are ignored, so cv adding an
  optional key needs no change here. The sections cv appends after `contact` (`volunteering`,
  `awards`, `exam_results`, `courses`, `publications`, `patents`, `organisations`) and the keys
  `grade`, `location` and `employment_type` are dropped until a page needs one: add the typed
  field to `internal/content/artefact.go` then. `certificates[].valid_until` is nullable.
- GitHub is called with `net/http`, 3 calls per repository (repository, languages, latest
  release), only for repositories named in the artefact. No listing call exists. A name must
  pass `ValidateRepo` (GitHub's owner rule, a repository name rule, no `.` or `..` segment) and
  each path segment is escaped. The response's `full_name` must equal the requested name, case
  insensitively: a renamed or transferred repository fails the build instead of showing its new
  owner silently. Public data needs no token scope: `GITHUB_TOKEN` is read from the environment
  when set, CI sets it from `secrets.GITHUB_TOKEN` on the content step only, and the Dockerfile
  has no build argument for it. It is sent as a bearer header and appears in no error, log line
  or output, which the tests assert. The `publish.yaml` image job holds `packages: write`, so
  that step's token carries it too: the step-scoped environment is the only limit, and the
  content step runs no code beyond `cmd/content`. A rate limit (429, or 403 with `Retry-After`
  or an exhausted quota) is waited out up to two minutes and fails beyond that. A 5xx or a
  timeout makes up to 4 attempts, so 3 retries, after a 1, 2 and 4 second backoff. The whole
  GitHub phase has a 3 minute deadline and fails with a clear message, so a slow API never ends
  as a CI job timeout.
- Redirects: both clients follow at most 5 hops, to https only. The API client follows only to
  the API host. The release download follows only to the host it started at and to
  `ReleaseRedirectHosts` (`github.com`, `objects.githubusercontent.com`,
  `release-assets.githubusercontent.com`). The current CDN names were not verified live: if
  GitHub moves its asset CDN the build fails with a refused redirect naming the host, and that
  variable is the one place to extend after checking. `Authorization` is dropped on any host
  change, and the download never sends the token at all: the release assets are public.
- Schema check of the merged dataset, decided: no JSON Schema file. The dataset is written from
  Go types, and `ParseDataset` reads it back strictly (unknown key, wrong version, a project
  without GitHub data and a stale flag without a reason all fail) on the bytes `Generate` just
  produced, so a dataset that fails never reaches the binary. The same strict read runs at start.
  `testdata/dataset.golden.json` pins the shape.
- Failure policy, decided. An artefact that fails to download or parse, or carries an unknown
  version, always fails the build. A failed GitHub call fails the build by default, because a
  silent partial dataset is the worst outcome. A rebuild may opt in with
  `-allow-stale -previous <earlier dataset.json>`: then all GitHub data comes from that earlier
  dataset, only if its `source.github.fetched_at` (not `generated_at`) is at most 14 days old and
  not in the future, and it covers every repository, and the dataset says so
  in `source.github.stale`, `stale_reason` and the original `fetched_at`. The fallback is all or
  nothing. No workflow enables it today, because no earlier dataset is kept anywhere yet. A
  scheduled rebuild that wants it has to supply one, for example from a cache.
- Determinism: `Generate` takes a clock. The same artefacts, the same GitHub answers and the same
  clock give the same bytes. `cmd/content` reads `SOURCE_DATE_EPOCH` to fix the clock. Without
  it, `generated_at` and `source.github.fetched_at` are the build time.
- `task content:build` skips while `cv-release.env` and the output are unchanged. Refresh the
  GitHub data of an unchanged pin with `task content:build --force`.
- `go build` and `go test` work on a clean checkout: `internal/content/generated/` is committed
  with a `.gitkeep` only, and `all:` makes `go:embed` match it. The tests read
  `internal/content/testdata/artefact/cv.{de,en}.json`, copies of cv's golden render fixtures with
  fake data, and a fake GitHub server. No test uses the network.

### A cv release does not reach the site by itself

The dataset is built, not fetched at runtime. A new cv release reaches the site only when this
repository rebuilds with the new tag and digests in `cv-release.env`. A person bumps that pin, see
above. Publishing a release of this repository then builds and
pushes the image (`publish.yaml`). Nothing in the cv repository triggers a rebuild yet: a
`repository_dispatch` from cv's release workflow is not set up and is out of scope here. GitHub
data (stars, releases, pushes) is as old as the last build for the same reason.

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

- Projects, pulled from the GitHub API and merged with hand written descriptions from the `cv`
  repository.
- Skills and tools, fed from the same `cv` repository.
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
