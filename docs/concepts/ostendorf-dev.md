# ostendorf.dev content

## Problem

- The repository is a govue scaffold: embedded SPA, admin listener, chart, CI. No content.
- `old_ostendorf.dev` is a static link page: GitHub, LinkedIn, email, nothing else.
- Projects, skills and tools exist only as hand written text in the `cv` repository and
  as repositories on GitHub, not as anything a visitor to this site can see.
- There is no contact channel on this domain; a visitor has to already know an email address.
- There is no page for setup, tools or homelab.

## Goals

- A visitor sees Simon's projects, each with GitHub metadata and a hand written description.
- A visitor sees skills and tools, sourced from the `cv` repository.
- A visitor can send a message through a contact form, delivered over SMTP.
- A visitor sees a uses page: setup, tools, homelab.
- No request the site serves causes the visitor's browser to contact any third party, or
  to receive any cookie set by anyone but this site. This holds for every page, not only
  the contact form.
- Project and skill pages stay crawlable without running JavaScript.

## Non goals

- Editing CV or project description text in this repository; that stays in `cv`.
- A CMS or an admin UI for content; content is data pulled in, not authored here.
- Matching simon-ostendorf.de's or ostendorf.tech's own design system or copy.
- Session based features beyond the contact form (no login, no profile).

## Approach

- Data pipeline, build time: the build fetches the `cv` repository's JSON release
  artefact and the GitHub API for the project list, merges them into one dataset, and
  the Go binary embeds that dataset the same way it already embeds `dist/`. Neither the
  running server nor the visitor's browser calls GitHub or the `cv` service at request
  time. Recommendation: build time, not the `cv` service's runtime API. Reasons:
  - It keeps the already-working "one self-contained binary" shape: no new runtime
    dependency, no new failure mode, no new timeout and cache plumbing like
    `ostendorf.tech`'s `internal/feedclient` needed for the Instagram feed.
  - It keeps the data prerenderable and crawlable (see below); `ostendorf.tech`
    deliberately left the Instagram section out of prerendering because it is runtime
    data with nothing worth indexing, but projects are exactly the content this site
    exists to have indexed.
  - The cost is staleness: a `cv` release does not reach the site until the site
    rebuilds. A `repository_dispatch` from `cv`'s own release workflow can trigger that
    rebuild automatically; this needs its own look at `cv`'s CI, out of scope here.
- GitHub refresh without a visitor's browser involved: the same build-time step calls
  the GitHub API server side (in CI, or a scheduled rebuild) and writes the merged,
  embedded dataset; the deployed binary never holds a GitHub token and never makes the
  call itself.
- Prerendering: recommend adopting `ostendorf.tech`'s pattern (`internal/routes`,
  `task prerender`, embedded `prerendered/`) for the projects, skills and uses pages,
  since build time is also when the data is current. A rebuild refreshes data and
  prerendered HTML together, so there is no extra staleness from prerendering itself.
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
  - simon-ostendorf.de is the neutral hub and renders the CV page; it stays the single
    place for the short, identity-level summary and the outbound links to the other two.
  - ostendorf.tech is Simon Ostendorf Veranstaltungstechnik, a separate business brand;
    nothing about event technology belongs here.
  - ostendorf.dev carries the deep, developer facing content only: the full project
    list, skills and tools, the uses page and this contact form. It does not restate the
    CV page's own summary.

## Alternatives considered

- Fetch `cv` and GitHub data at runtime from the site's own process (a `Syncer` like
  `ostendorf.tech`'s `internal/feedclient`, polling the `cv` API and the GitHub API on
  an interval). Dropped: couples this site's availability to the `cv` service and to
  GitHub's API and its rate limit, and reintroduces the "unprerendered, not crawled"
  trade-off `ostendorf.tech` accepted only for a section with nothing worth indexing.
- A separate internal "projects" service in front of GitHub, mirroring `ostendorf.tech`'s
  `cmd/feed` split. Dropped for now: that split earned its cost by moving a long lived
  Meta access token out of the public facing binary; a build-time GitHub API call holds
  no comparable secret, so the split has no matching benefit here.
- Prerendering only the static pages (uses, home) and rendering projects and skills
  client side from an embedded JSON. Dropped: it would leave the main reason to visit
  the site uncrawlable, the opposite of the Instagram case this pattern is borrowed from.

## Open questions

- Does `cv`'s release workflow get a `repository_dispatch` (or similar) into this
  repository so a new release rebuilds the site automatically? Owner: whoever specs the
  `cv` repository's release pipeline next.
- Where do SMTP credentials and the recipient address live operationally (which secret
  store, which chart values)? Owner: next milestone that implements the contact form.
- Does the merged build-time dataset need a schema check against `cv`'s JSON release
  artefact, and where does that check live? Owner: whoever implements the data pipeline.
- Should ostendorf.dev link to simon-ostendorf.de and ostendorf.tech in its own nav or
  footer, and is there a shared convention for that across all three sites? Owner: not
  yet assigned.
- What happens to the projects and skills pages when the GitHub API call fails during a
  build: fail the build, or ship the last known good dataset? Owner: whoever implements
  the data pipeline.
