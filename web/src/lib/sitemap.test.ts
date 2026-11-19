import { describe, expect, it } from 'vitest'

import { routeTable } from './routeTable'
import { sitemapXml } from './sitemap'

describe('sitemapXml', () => {
  const xml = sitemapXml(routeTable, 'https://ostendorf.dev')

  it('lists the home page with a slash and the legal pages without one', () => {
    expect(xml).toContain('<loc>https://ostendorf.dev/</loc>')
    for (const page of ['de/imprint', 'en/imprint', 'de/privacy', 'en/privacy']) {
      expect(xml).toContain(`<loc>https://ostendorf.dev/${page}</loc>`)
      expect(xml).not.toContain(`<loc>https://ostendorf.dev/${page}/</loc>`)
    }
  })

  it('lists no terms page, the site has no terms', () => {
    expect(xml).not.toContain('terms')
    expect(xml).not.toContain('agb')
  })

  it('lists one url per route', () => {
    expect(xml.match(/<url>/g)).toHaveLength(routeTable.length)
  })
})
