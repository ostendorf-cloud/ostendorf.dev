# 0001 Read the cv data from cvd at runtime

- Status: accepted
- Date: 2026-11-27
- Concept: [ostendorf.dev content](../concepts/ostendorf-dev.md)

## Context

- The owner decided on 2026-11-20 that the sites read `cvd` at runtime, server to server. The
  reasons: no stale artefact in a site, and `cvd` can sit behind a firewall.
- The owner added decisions on 2026-11-27: no background timer, no token, no hard coded
  address, and no failed readiness probe when `cvd` is down.
- `cvd` at `cv` commit `ba53db4` serves one JSON document per language (`de`, `en`) with
  `schema_version` 1, a strong `ETag` and 304 answers. It carries no GitHub data and reads its
  data once at start.
- This site runs as one replica. A copy held in its process is lost at a restart.
- The visitor's browser must never contact `cvd` or any other third party.

## Decision

The Go server reads the cv document from `cvd` on request into an in-memory cache, one entry
per language, and serves the last good copy when `cvd` fails. The reasons follow as policy.

- Fetch on request, no timer. The first request after the cache expired triggers one refresh,
  and concurrent requests share it. A request that finds an expired copy serves it and does
  not wait.
- Only a request with no copy at all waits, bounded by `CVD_TIMEOUT`. If that read fails, the
  data pages answer `503` with `Retry-After: 30`. Readiness never depends on `cvd`.
- After a failed read the next attempt waits at least 30 seconds, also when no copy exists.
  The gap is kept per language. A request inside the gap answers at once and calls nothing.
- The last good copy never expires and lives in memory only. A document that does not decode,
  names another `language` or carries an unknown `schema_version` counts as a failed read.
- `cvd` loads the GitHub data of each project itself. This site makes no GitHub call and
  holds no token.
- The Go client is a public package in the `cv` repository. This site imports it and holds no
  client of its own.

## Consequences

- Good: a `cvd` outage after one read changes no page, and the probes stay green.
- Good: no GitHub token and no GitHub rate limit in this site.
- Bad: with `cvd` down at start, the data pages show the `503` placeholder until a read works.
  A restart during an outage loses the copy.
- Bad: the data pages cannot be prerendered with their data. The Go server renders them
  per request, and how it combines with the prerendered shell is open.
- Bad: the code round waits for two changes in `cv`, GitHub enrichment in `cvd` and the
  public client package. This site then depends on that package's release cadence.
- Bad: the cache lifetime and the fact that `cvd` reads once at start bound how fresh a page is.

## Alternatives considered

- Build time merge of the release artefact and the GitHub API, embedded in the binary. Lost:
  the site would carry a stale artefact until its next rebuild, and it needed a dispatch from `cv`.
- A background poller like `ostendorf.tech`'s `internal/feedclient`. Lost: the owner chose
  fetch on request with no timer.
- A failed readiness probe until the first read. Lost: a down `cvd` would take the whole site
  out of rotation, including the legal pages.
- The visitor's browser calling `cvd`. Lost: it breaks the no third party property.
- This site calling GitHub itself, or a separate projects service in front of it. Lost: it
  needs a token or the unauthenticated rate limit, and `cvd` already loads the data once.
- A small client inside this repository. Lost: the owner builds the public package now.
