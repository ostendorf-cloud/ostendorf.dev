# ostendorf.dev

The developer site of Simon Ostendorf: projects, skills and contact.

The global baseline already applies here and is not repeated below. This succeeds
old_ostendorf.dev, a plain HTML and CSS link page with no JavaScript and no third party
request; this site keeps that no-third-party-request property while becoming a govue
application.

Detail per directory lives in a nested `AGENTS.md` that loads once a file below it is read.

## Layout

- `cmd/website` wires the service and embeds the built `dist/` and `prerendered/`. It also holds
  the tests over the built output: `cmd/website/AGENTS.md`.
- `internal/config` loads and validates the environment once: `internal/config/AGENTS.md`.
- `internal/content` reads the cv document from `cvd` at runtime, with a cache, the last good
  copy and a 503 placeholder: `internal/content/AGENTS.md`.
- `internal/webapp` builds the one HTTP handler, the analytics and access log chain and serves
  the legal pages: `internal/webapp/AGENTS.md`.
- `internal/routes` is the one route table, `routes.json`, read by Go and by the frontend:
  `internal/routes/AGENTS.md`.
- `internal/depsync` checks that the govue halves move together: `internal/depsync/AGENTS.md`.
- `web/` is the Vue SPA, the theme and the prerender step: `web/AGENTS.md`.
- `charts/ostendorf.dev/` is the Helm chart: `charts/ostendorf.dev/AGENTS.md`.
- `Dockerfile` and `docker-compose.yaml` build and run the one binary locally.
- `scripts/smoke-test-image.sh` starts a built image and asserts it answers health, readiness and
  metrics on the admin listener and a request on the public one; the `docker-build` job in
  `.github/workflows/checks.yaml` runs it right after building the image.

## Hard constraints

- A build needs the frontend first. `cmd/website` cannot compile against an empty `dist/` or
  `prerendered/`. Use `task build` or `task test:go`, not a bare `go build ./cmd/website`. See
  `cmd/website/AGENTS.md`.
- govue's two halves move together. `go.mod` and `web/package.json` must name the same govue
  tag, and `task check:govue-version` enforces it. See `internal/depsync/AGENTS.md`.
- The site links to neither simon-ostendorf.de nor ostendorf.tech in its navigation or footer.
  See `web/AGENTS.md`.
- The site makes no third party request from the browser. `TestBuiltOutputMakesNoThirdPartyRequest`
  proves it over the built output.

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

- The projects page, rendered from the `cvd` document with their hand written descriptions and
  the GitHub languages and latest release `cvd` holds. The client, the cache and the 503
  placeholder exist in `internal/content` and `internal/webapp`.
- The skills and tools page, rendered from the same document.
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
