import { routePath, type RouteEntry } from './routeTable'

/** sitemapXml is the sitemap of every route in table under origin: one url
 *  per route, in slug order. It is the same document internal/webapp serves
 *  at /sitemap.xml for the same origin (sitemap.go builds it from the same
 *  table); the prerender writes it so govue-check-legal can read what the
 *  server lists, and cmd/website's test compares the two byte for byte. */
export function sitemapXml(table: readonly RouteEntry[], origin: string): string {
  const base = origin.replace(/\/$/, '')
  const entries = [...table].sort((a, b) => (a.slug < b.slug ? -1 : a.slug > b.slug ? 1 : 0))
  const lines = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
  ]
  for (const entry of entries) {
    lines.push('  <url>', `    <loc>${base}${routePath(entry)}</loc>`, '  </url>')
  }
  lines.push('</urlset>', '')
  return lines.join('\n')
}
