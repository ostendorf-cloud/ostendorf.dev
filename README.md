# ostendorf.dev

The developer site of Simon Ostendorf: projects, skills and contact.

## What it does

A Go service serves a Vue 3 single page application built on the organisation's own `govue`
framework. This milestone ships one placeholder page carrying the real contact and profile
links from the predecessor link page; projects, skills and a contact form are later milestones,
see `AGENTS.md`.

## Install

```sh
task setup
```

Installs the pinned toolchain (Go, Node, pnpm, and the lint and release tooling) through `mise`.

## Use

```sh
task web:build   # builds the SPA into cmd/website/dist
task build        # compiles the website binary into bin/website
bin/website
```

`task build` also runs `task content:build`, which downloads the cv release pinned in
`cv-release.env` and calls the GitHub API, so it needs the network and a set pin. Until the pin is
set, it fails with "no cv release is pinned". `GITHUB_TOKEN` is optional locally. A binary needs
the generated `internal/content/generated/dataset.json` and refuses to start without it. For
`docker compose up --build`, run `task content:build` first: the image build refuses to run
without the dataset and never holds a token.

The server listens on `:8080` and answers `/healthz`, `/readyz` and `/metrics` on a second,
administrative listener at `:8081`, never on the public one. Both addresses are configurable
through `ADDR` and `ADMIN_ADDR`; see "Configuration" below.

## Configuration

`.env.example` lists every variable. All are optional, and none is a secret.

- `UMAMI_URL` and `UMAMI_WEBSITE_ID` turn on server side page view counting. Both or neither.
- `TRUSTED_PROXIES` names the proxies in front of the service, so the client address is known.

What is counted, and what is not, is in `docs/analytics.md`.

## Container

```sh
docker compose up --build
```

Builds the multi-stage `Dockerfile` (frontend, static Go binary, then a minimal non-root
runtime stage with no shell) and serves the site at `http://localhost:8080`.
`docker-compose.yaml` is for local development only; see `charts/ostendorf.dev/README.md` for a
cluster installation.

The published image is one manifest list covering `linux/amd64` and `linux/arm64`.

`scripts/smoke-test-image.sh <image-ref>` starts an already built image the same way and
asserts its admin listener (`:8081`) answers `/healthz`, `/readyz` and `/metrics`, its public
listener serves a request, and the public listener itself does not answer `/metrics`. The
`docker-build` job in `.github/workflows/checks.yaml` runs it against the image it just built,
so the container has started and served a real request at least once before anyone ever runs it
in the cluster.

## Development

```sh
task setup
task check
```

`task check` runs every lint, build and test target one after another; `task --list` shows them
individually. `task web:build` has to run before `task build` or `task test:go` can compile
`cmd/website`, since it embeds that output; `task build` and `task test:go` already declare that
dependency, so running either one through Task is enough.

Releases are driven by [release-me](https://github.com/ostendorf-cloud/release-me): Conventional
Commits decide the version, and merging the release pull request it maintains cuts the release.
`CHANGELOG.md` is generated, never hand edited.

## License

Proprietary, all rights reserved. See `LICENSE` and `NOTICE`.
