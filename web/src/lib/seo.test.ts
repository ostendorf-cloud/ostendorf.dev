import { describe, expect, it } from 'vitest'

import { routeTable } from './routeTable'
import { canonical, descriptions, NOT_FOUND, path, titles } from './seo'

// titles and descriptions are kept separate from routes.json
// deliberately (see seo.ts), which only stays safe as long as this test
// fails the moment the two lists disagree: a route with no entry, or an
// entry for a name that is not a real route.

describe('titles', () => {
  const wantNames = new Set([...routeTable.map((entry) => entry.name), NOT_FOUND])

  it.each(routeTable)('has a title for routes.json entry $name', (entry) => {
    expect(titles[entry.name]).toBeDefined()
  })

  it('has a title for the 404 page', () => {
    expect(titles[NOT_FOUND]).toBeDefined()
  })

  it('has no title for a name that is not a real route', () => {
    for (const name of Object.keys(titles)) {
      expect(wantNames.has(name), `titles names "${name}", which is not a real route`).toBe(true)
    }
  })

  it('carries exactly one title per real route plus the 404 page', () => {
    expect(Object.keys(titles)).toHaveLength(wantNames.size)
  })
})

describe('descriptions', () => {
  const wantNames = new Set([...routeTable.map((entry) => entry.name), NOT_FOUND])

  it.each(routeTable)('has a description for routes.json entry $name', (entry) => {
    expect(descriptions[entry.name]).toBeDefined()
  })

  it('has a description for the 404 page', () => {
    expect(descriptions[NOT_FOUND]).toBeDefined()
  })

  it('has no description for a name that is not a real route', () => {
    for (const name of Object.keys(descriptions)) {
      expect(wantNames.has(name), `descriptions names "${name}", which is not a real route`).toBe(
        true,
      )
    }
  })

  it('carries exactly one description per real route plus the 404 page', () => {
    expect(Object.keys(descriptions)).toHaveLength(wantNames.size)
  })
})

describe('path and canonical', () => {
  it.each(routeTable)('resolves routes.json entry $name to its own slug', (entry) => {
    const want = entry.slug === '' ? '/' : `/${entry.slug}/`
    expect(path(entry.name)).toBe(want)
  })

  it.each(routeTable)(
    'builds a canonical URL for routes.json entry $name without throwing',
    (entry) => {
      expect(() => canonical(entry.name)).not.toThrow()
    },
  )

  it('resolves the 404 page to the root path', () => {
    expect(path(NOT_FOUND)).toBe('/')
  })

  it('builds a canonical URL for the 404 page without throwing', () => {
    expect(() => canonical(NOT_FOUND)).not.toThrow()
  })
})
