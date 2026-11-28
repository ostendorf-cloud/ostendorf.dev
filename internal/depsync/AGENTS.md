# internal/depsync

- `internal/depsync` checks that `go.mod` and `web/package.json` pin the same govue version, and
  that `web/pnpm-lock.yaml` still resolves govue's Vue package to its `web/` subdirectory.

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
