# ostendorf.dev content

## Problem

- The repository is a govue scaffold: embedded SPA, admin listener, chart, CI. No content.
- `old_ostendorf.dev` is a static link page: GitHub, LinkedIn, email, nothing else.
- Projects, skills and tools exist as hand written data in the `cv` repository and as
  repositories on GitHub. A visitor to this site sees none of it.
- The `cv` repository now serves that data through its service `cvd`. This site has no code
  that reads it.
- There is no contact channel on this domain. A visitor has to already know an email address.
- There is no page for setup, tools or homelab.

## Goals

- A visitor sees Simon's projects, each with a hand written description and, where GitHub
  has them, its languages and latest release, all read from `cvd`.
- A visitor sees skills and tools, read from `cvd` at runtime.
- A visitor sees a uses page: setup, tools, homelab.
- A visitor can send a message through a contact form, delivered over SMTP.
- A `cvd` outage after one successful read changes no page and fails no probe.
- The readiness probe never depends on `cvd`.
- No request the site serves causes the visitor's browser to contact any third party, or
  to receive any cookie set by anyone but this site. This includes `cvd`.
- Project and skill pages stay crawlable without running JavaScript.

## Non goals

- Editing CV or project description text in this repository. That stays in `cv`.
- A CMS or an admin UI for content. Content is data read in, not authored here.
- The changes to the `cv` repository and to `cvd`. They are prerequisites of this work, built
  and tracked there.
- A page that renders the data (issues 5 and 6) and the PDF download.
- A background timer, a disk copy of the document, a GitHub call from this site or a token.
- Matching simon-ostendorf.de's or ostendorf.tech's own design system or copy.
- Session based features beyond the contact form (no login, no profile).

## Approach

### Reading the data from `cvd`

- Decided by the owner on 2026-11-20, recorded in `cv`'s concept doc: the sites read `cvd`
  at runtime, server to server. Reasons: no stale artefact in a site, and `cvd` can sit
  behind a firewall.
- Decided by the owner on 2026-11-27:
  - The site fetches on request into a cache. The first request after the cache expired
    triggers the read. No background timer. The cache lifetime is configurable with a
    documented default.
  - Before the first successful read the site serves a placeholder page. It does not fail
    its readiness probe. Readiness does not depend on `cvd`.
  - Server to server only. The visitor's browser never calls `cvd`. No token. The base URL
    is an environment variable exposed by the chart, later `https://cv.ostendorf.systems`.
    No address is hard coded.
  - The document is validated on read. An unknown schema version is refused with a clear
    log line, and the last good copy stays in use. The client sets a timeout.
- Decided by the owner on 2026-11-27, after the concept step:
  - `cvd` loads the GitHub data of each project (languages, latest release) itself and caches
    it. The site gets it from `cvd` and needs no GitHub token and no GitHub call.
  - The Go client for `cvd` becomes a public package in the `cv` repository, built now. This
    site imports it and holds no client of its own.
  - The site requests `de` and `en` and caches each language on its own.
  - Before the first successful read the data pages answer `503` with `Retry-After`.
  - The defaults in "Configuration and behaviour" are accepted.
  - The privacy page gets one sentence on the server to server call to `cvd`.
- The Go server renders every page that shows the data, from the cached document. The
  browser never holds the data source.

### Prerequisites in the `cv` repository

- GitHub enrichment in `cvd`: `cvd` fetches `languages` and `latest_release` per project and
  caches them, so `GET /api/v1/cv` carries them. Today it does not, see the next block.
- A public Go client package for `cvd`, with the timeout, the schema version check and the
  `ETag` revalidation described below.
- The code round of this issue waits for both. This step writes documentation only.

### What `cvd` serves

Read at `cv` `origin/main` (commit `ba53db4`), not from memory. This is the state before the
prerequisites above.

- `GET /api/v1/cv` answers one document per language (`internal/cvd/cvd.go:173`, `getCV` at
  `:253`). The query parameter `lang` takes `de` or `en` (`internal/cv/render_json.go:22`).
  The default is `en` (`cvd.go:30`). An unknown value is a 400 (`cvd.go:226-233`).
- The document carries the integer `schema_version`, currently `1`
  (`render_json.go:19` and `:38`, `cv` `README.md:93`). It also carries `language`.
- The `cv` `README.md:98` says a consumer refuses a `schema_version` it does not know. A
  new key a consumer can ignore does not raise the version. The client must therefore
  tolerate unknown keys and refuse only an unknown version.
- The document does not carry `languages` or `latest_release` per project. `cvd` renders
  with `cv.RenderJSON` (`cvd.go:103`), which merges nothing from GitHub. The code says so:
  "cvd calls RenderJSON, never this, so the service makes no GitHub call"
  (`render_json.go:268`). The `cv` `README.md:105-111` agrees. A project entry from `cvd` has
  `id`, `github_repo` and `description` only (`render_json.go:195-199`).
- The enriched fields exist only in the release artefact, which `cv render json` writes through
  `RenderJSONEnriched` (`cmd/cv/main.go:200`).
- `getCV` sets `Cache-Control: no-cache` and a strong `ETag`, and answers a matching
  `If-None-Match` with a 304 (`cvd.go:272-273` and `:316-325`). A refresh after expiry
  therefore costs a 304 while nothing changed.
- `cvd` reads its data once at start (`cvd.go:58-59`). The document changes only when `cvd`
  restarts. A short cache lifetime here mostly yields 304 answers.
- `cvd` also serves `GET /api/v1/cv.pdf` today (`cvd.go:174`, `getPDF` at `:281`). It answers
  503 with the code `pdf_unavailable` when no PDF is loaded (`cvd.go:303-305`). This site
  does not use the route in this milestone.
- Not established: the final public URL of `cvd`. The `cv` chart's default host is
  `cv.ostendorf.cloud` (`charts/cv/values.yaml:67`), the issue text names
  `cv.ostendorf.systems`. See the open questions.

### Cache and fallback

- One cached document per language, held in memory, with the time of its last good read.
- A request that finds a fresh copy serves it. No call to `cvd`.
- A request that finds an expired copy triggers one refresh. Concurrent requests share it.
  They do not start a second call.
- A request that triggers a refresh while an expired copy exists serves that copy and does
  not wait. Only a request with no copy at all waits, bounded by `CVD_TIMEOUT`.
- A failed refresh keeps the last good copy and logs the failure. The copy is served until
  a later refresh succeeds. It never expires on its own.
- After a failed refresh the next attempt waits at least 30 seconds, so a down `cvd` does not
  get one call per request.
- A document with an unknown `schema_version` is treated like a failed refresh, with its
  own log line that names the version found and the versions supported.
- With no copy at all, the data pages show the placeholder, and the next request tries again.

### Pages, prerendering and the rest

- Prerendering: the data pages can no longer be prerendered with their data at build time.
  The Go server renders them per request. Static pages (home, uses, legal, 404) stay
  prerendered through `internal/routes` and `task prerender`. How the Go server and the
  prerendered shell combine is for issues 5 and 6.
- Contact form, ported from `ostendorf.tech`/`old_simon-ostendorf.de`, needs pieces this
  scaffold does not have yet:
  - `internal/pow` (challenge issue and verify) and `internal/contact` (validate, build
    mail, send over SMTP).
  - SMTP and recipient configuration in `internal/config`, with the `_FILE` secret
    convention the sibling sites use.
  - A session manager and CSRF protector wired into a new `internal/webapp`-shaped
    handler, govue's `session.MemoryStore` and `csrf` package, not present in
    `cmd/website/main.go` today.
  - `GET /api/v1/challenge` and `POST /api/v1/contact` endpoints.
  - `web/src/lib` challenge client and `web/src/workers/pow.worker.ts`, the Web Worker
    that solves the challenge, cookie free and third party free, same as the sibling sites.
  - Single replica and `Recreate` rollout in the chart, since the in-process session
    store does not survive two pods, same constraint `ostendorf.tech` carries.
- Boundary against the sibling sites:
  - simon-ostendorf.de is the neutral hub and renders the CV page. It stays the single
    place for the short, identity-level summary and the outbound links to the other two.
  - ostendorf.tech is Simon Ostendorf Veranstaltungstechnik, a separate business brand.
    Nothing about event technology belongs here.
  - ostendorf.dev carries the deep, developer facing content only: the full project
    list, skills and tools, the uses page and this contact form. It does not restate the
    CV page's own summary.

## Configuration and behaviour

Decided by the owner on 2026-11-27: the defaults below, the placeholder with `503` and the
behaviour without `CVD_URL`. The names follow `internal/config/config.go` and `.env.example`:
upper case, the service prefix first, durations as Go duration strings, like
`UMAMI_TIMEOUT=5s`.

| Setting | Default | Notes |
| --- | --- | --- |
| `CVD_URL` | none | Base URL of `cvd`, no path, no credentials. |
| `CVD_CACHE_TTL` | `5m` | Age after which a request triggers a refresh. |
| `CVD_TIMEOUT` | `5s` | Bound of one call to `cvd`, 1s to 30s as `UMAMI_TIMEOUT`. |

- The client appends `/api/v1/cv` and `?lang=` itself. The base URL carries no path.
- The chart exposes the three values under `config.cvd` (`url`, `cacheTtl`, `timeout`). It
  renders each one only when set, as it does for Umami. `values.schema.json` checks the URL.
- `CVD_URL` unset: the data pages serve the placeholder, the start logs one info line, and
  the site still starts. A local run needs no `cvd`.
- The cache lifetime default of five minutes is short enough to show a `cvd` restart soon.
  It is long enough to keep the call rate near zero. A refresh after expiry sends
  `If-None-Match` with the last `ETag`.
- Placeholder: a short page in the site's shell that says the data is not available yet. It
  appears on the data pages only. Home, uses, legal pages and the probes are unaffected.
  Status code: `503` with `Retry-After`, so a crawler does not index the placeholder.
- The retry gap after a failed refresh is 30 seconds, see "Cache and fallback".
- The site requests `de` and `en` and keeps one copy per language.
- The fallback copy lives in memory only. A restart of the site while `cvd` is down brings
  the placeholder back.
- Log lines: one on a failed read (error, age of the copy served), one on a refused schema
  version (version found, versions supported), one when a read succeeds after a failure.
- The document is validated on read: it must decode, `schema_version` must be supported,
  and `language` must equal the language asked for.

## Alternatives considered

- Build time merge, the first plan: the build fetches the `cv` JSON release artefact and the
  GitHub API, merges them and embeds the dataset in the binary. Replaced on 2026-11-20 by the
  owner's decision. A site would carry a stale artefact until its next rebuild. It also needed
  a `repository_dispatch` from `cv` to stay current, and the owner wants the CV API behind
  his firewall.
- A background poller (a `Syncer` like `ostendorf.tech`'s `internal/feedclient`). Dropped on
  2026-11-27: the owner chose fetch on request with no background timer.
- Failing the readiness probe until the first read. Dropped on 2026-11-27: a down `cvd` would
  take the whole site, including the legal pages and the contact form, out of rotation.
- The visitor's browser calling `cvd`. Dropped: it breaks the no third party promise and
  puts `cvd` on the open web.
- This site calling GitHub itself for languages and latest release. Dropped on 2026-11-27: it
  needs a token or lives with the unauthenticated rate limit, and `cvd` loads the data once
  for every site that reads it.
- A separate internal "projects" service in front of GitHub, mirroring `ostendorf.tech`'s
  `cmd/feed` split. Dropped: that split moved a long lived token out of the public facing
  binary, and this site holds no token. `cvd` already is that service.
- A small client written inside this repository. Dropped on 2026-11-27: the owner builds a
  public client package in the `cv` repository now, and this site imports it.
- Prerendering only the static pages and rendering projects and skills client side from an
  embedded JSON. Dropped: it leaves the main reason to visit the site uncrawlable.

## Open questions

- How do the Go server and the prerendered shell combine on the data pages? Owner: issue 5.
- What is the final public URL of `cvd`? Owner: the site owner. It sets the chart value.
- Which host is right, `cv.ostendorf.cloud` (the `cv` chart default) or
  `cv.ostendorf.systems` (the issue text)? Owner: the site owner.
- How does `cvd` reload its data without a restart? It reads once at start today. Owner: the
  `cv` repository's sync work, see `cv`'s concept doc.
- Where do SMTP credentials and the recipient address live operationally (which secret
  store, which chart values)? Owner: next milestone that implements the contact form.
- Should ostendorf.dev link to simon-ostendorf.de and ostendorf.tech in its own nav or
  footer, and is there a shared convention for that across all three sites? Owner: not
  yet assigned.
