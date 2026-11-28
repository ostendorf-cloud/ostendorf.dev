# internal/config

- `internal/config` loads and validates the website's own environment once, aggregated into
  one error. `SiteURL` (`SITE_URL`, default `https://ostendorf.dev`) is the base
  `internal/webapp`'s `robots.txt` and `sitemap.xml` build every URL from. There is no
  secret-shaped variable yet: no session, no OIDC, no CSRF and no form exist in this milestone.
- The analytics variables (`UMAMI_URL`, `UMAMI_WEBSITE_ID`, `TRUSTED_PROXIES`) are described in
  `internal/webapp/AGENTS.md`. The cvd variables (`CVD_URL`, `CVD_CACHE_TTL`, `CVD_TIMEOUT`) are
  described in `internal/content/AGENTS.md`.
