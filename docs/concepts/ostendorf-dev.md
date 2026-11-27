# ostendorf.dev content

## Problem

- The repository is a govue scaffold: embedded SPA, admin listener, chart, CI. No content.
- `old_ostendorf.dev` is a static link page: GitHub, LinkedIn, email, nothing else.
- Projects, skills and tools exist as hand written data in the `cv` repository and as
  repositories on GitHub. A visitor to this site sees none of it.
- The `cv` repository serves that data through its service `cvd`. This site has no code
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
- No request the site serves causes the visitor's browser to contact any third party, or
  to receive any cookie set by anyone but this site. This includes `cvd`.
- Project and skill pages stay crawlable without running JavaScript.

## Non goals

- Editing CV or project description text in this repository. That stays in `cv`.
- A CMS or an admin UI for content. Content is data read in, not authored here.
- The changes to the `cv` repository and to `cvd`. They are prerequisites, built there.
- A page that renders the data (issues 5 and 6) and the PDF download.
- A background timer, a disk copy of the document, a GitHub call from this site or a token.
- Matching simon-ostendorf.de's or ostendorf.tech's own design system or copy.
- Session based features beyond the contact form (no login, no profile).

## Approach

- The Go server reads the cv document from `cvd` at runtime, server to server, into a cache
  with the last good copy as fallback. [ADR 0001](../adr/0001-read-cv-data-from-cvd-at-runtime.md)
  records the decision and the cache, retry and placeholder policy.
- Two prerequisites in the `cv` repository come first: GitHub enrichment in `cvd`, and a public
  Go client package that this site imports. The code round of issue 4 waits for both.
- The Go server renders every page that shows the data, from the cached document. Static pages
  (home, uses, legal, 404) stay prerendered through `internal/routes` and `task prerender`.
  How the Go server and the prerendered shell combine is for issue 5.
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

Decided by the owner on 2026-11-27. The names follow `internal/config/config.go` and
`.env.example`: upper case, the service prefix first, durations as Go duration strings.

| Setting | Default | Notes |
| --- | --- | --- |
| `CVD_URL` | none | Base URL of `cvd`, no path, no credentials. |
| `CVD_CACHE_TTL` | `5m` | Age after which a request triggers a refresh. |
| `CVD_TIMEOUT` | `5s` | Bound of one call to `cvd`, 1s to 30s as `UMAMI_TIMEOUT`. |

- The chart exposes the three values under `config.cvd` (`url`, `cacheTtl`, `timeout`). It
  renders each one only when set, as it does for Umami. `values.schema.json` checks the URL.
- Without `CVD_URL` the site starts, logs one info line and serves the placeholder on the data
  pages. A local run needs no `cvd`.
- The placeholder is a short page in the site's shell that says the data is not available
  yet. Home, uses, legal pages and the probes are unaffected.
- The site requests `de` and `en` and keeps one copy per language.
- The log carries one line per failed read, per refused schema version and per recovery.
- The privacy page gets one sentence on the server to server call to `cvd`.

## Alternatives considered

- The alternatives for reading the data are in ADR 0001: build time merge, a background
  poller, a failed readiness probe, the browser calling `cvd`, and a client of this site's own.
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
