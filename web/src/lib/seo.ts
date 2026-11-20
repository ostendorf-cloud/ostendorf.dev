import { routeTable } from './routeTable'

/** The site's canonical origin. Overridable through VITE_SITE_URL for a
 *  staging build; the default matches the Go service's own default
 *  (internal/config.DefaultSiteURL). */
export const SITE_URL = import.meta.env.VITE_SITE_URL ?? 'https://ostendorf.dev'

/** NOT_FOUND is the 404 page's own pseudo route: it answers no path of
 *  its own, so it is deliberately not part of internal/routes' table
 *  (see internal/routes/routes.go) and is added here, and in
 *  router/index.ts as NOT_FOUND_ROUTE_NAME, by hand instead of being
 *  derived from it. */
export const NOT_FOUND = 'notFound' as const

/** PageRoute is every route this milestone prerenders: every name
 *  routes.json carries, plus the 404 page. A plain string rather than a
 *  literal union: routes.json is read at build time, not known to the
 *  type checker, so the real safety net is seo.test.ts, which checks
 *  every name routeTable actually carries against titles and
 *  descriptions below, in both directions. */
export type PageRoute = string

/** slugs maps a route to its path segment, derived from routeTable so a
 *  page added there needs no edit here. The 404 page has no segment of
 *  its own, the same empty value the home page's own slug happens to
 *  share. */
const slugs: Record<string, string> = Object.fromEntries(
  routeTable.map((entry): [string, string] => [entry.name, entry.slug]),
)
slugs[NOT_FOUND] = ''

/** path is the absolute path of route, always starting and ending with
 *  a slash. */
export function path(route: PageRoute): string {
  const slug = slugs[route]
  return slug === '' ? '/' : `/${slug}/`
}

/** canonical is the absolute URL of route. */
export function canonical(route: PageRoute): string {
  return SITE_URL + path(route)
}

export interface PageMeta {
  title: string
  description: string
}

/** titles and descriptions name each route's title and meta description.
 *  Kept separate from routes.json deliberately: routes.json is embedded
 *  into the Go binary via go:embed and read by Go code that has no use
 *  for page copy; coupling the Go-owned route table's schema to
 *  frontend-only text would leak a frontend concern into a shared,
 *  Go-consumed resource. Exported so seo.test.ts can check they cover
 *  exactly routeTable's names plus NOT_FOUND. */
export const titles: Record<string, string> = {
  home: 'Simon Ostendorf',
  projects: 'Projects',
  notFound: 'Page not found',
}

export const descriptions: Record<string, string> = {
  home: 'Simon Ostendorf — DevOps & Cloud Engineer. Projects, skills and contact.',
  projects:
    'The projects of Simon Ostendorf, each with its description, languages and latest release.',
  notFound: 'The page you were looking for does not exist.',
}

/** pageMeta resolves the title and meta description of route. */
export function pageMeta(route: PageRoute): PageMeta {
  return { title: titles[route], description: descriptions[route] }
}

/** fullTitle is the browser tab title: the page's own title for the home
 *  page, and "<title> – Simon Ostendorf" for every other page, matching
 *  the predecessor site's own convention. */
export function fullTitle(route: PageRoute, meta: PageMeta): string {
  return route === 'home' ? meta.title : `${meta.title} – Simon Ostendorf`
}

// Project pages. Their paths and texts come from the generated dataset, not
// from routes.json, so they are not in the maps above.

/** PROJECT is the route name of a project page: pass it to fullTitle. */
export const PROJECT = 'project' as const

/** MAX_DESCRIPTION is where a meta description is cut. */
const MAX_DESCRIPTION = 155

/** projectPath is the absolute path of the page of project id. */
export function projectPath(id: string): string {
  return `/projects/${id}/`
}

/** projectCanonical is the absolute URL of the page of project id. */
export function projectCanonical(id: string): string {
  return SITE_URL + projectPath(id)
}

/** projectMeta is the title and meta description of one project page: the
 *  project's name, and its hand written description cut at a word boundary,
 *  or a plain sentence when it has none. */
export function projectMeta(project: { name: string; description: string | null }): PageMeta {
  const text = project.description ?? `${project.name}, a project of Simon Ostendorf.`
  if (text.length <= MAX_DESCRIPTION) {
    return { title: project.name, description: text }
  }
  const cut = text.slice(0, MAX_DESCRIPTION)
  const end = cut.lastIndexOf(' ')
  return { title: project.name, description: `${cut.slice(0, end > 0 ? end : MAX_DESCRIPTION)}...` }
}
