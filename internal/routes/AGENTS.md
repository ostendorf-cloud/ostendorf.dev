# internal/routes

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
