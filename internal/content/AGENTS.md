# internal/content

- `internal/content` reads the cv document from `cvd` at runtime through the public client of the
  `cv` module (`github.com/ostendorf-cloud/cv/client`, a pseudo-version until `cv` tags a
  release). Config (`CVD_URL`, `CVD_CACHE_TTL`, `CVD_TIMEOUT`) lives in `internal/config`, the
  `Store` is built in `cmd/website/main.go` from one client per language (`de`, `en`) and handed
  to `internal/webapp` as `Config.Content`. ADR 0001 is the policy: fetch on a request, a copy per
  language, one shared refresh after the cache age, the old copy served meanwhile, a 30s gap after
  a failed read, the last good copy kept in memory. Without a copy a data page answers 503 with
  `Retry-After: 30` and the prerendered `unavailable.html`, built like the 404 page and not a
  `routes.json` entry. A change in `cv` reaches the site after the cache age, not after a
  rebuild. Readiness never depends on `cvd`. `webapp`'s `dataHandler` wraps a data page's
  renderer and is unused until the projects and skills pages exist.
