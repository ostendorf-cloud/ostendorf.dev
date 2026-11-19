import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import { createSiteI18n } from '../lib/language'
import AppFooter from './AppFooter.vue'

describe('AppFooter', () => {
  it('carries a copyright note for the current year', () => {
    render(AppFooter)

    expect(screen.getByText(`© ${new Date().getFullYear()} Simon Ostendorf`)).toBeVisible()
  })

  // See AGENTS.md: simon-ostendorf.de is the hub that links out to the
  // other two sites, so this footer deliberately carries none of its
  // own.
  it('carries no link to a sibling site, only the legal pages', () => {
    render(AppFooter)

    const hrefs = screen.getAllByRole('link').map((link) => link.getAttribute('href'))
    expect(hrefs).toEqual(['/en/imprint', '/en/privacy'])
  })

  it('links the legal pages of the active language', () => {
    render(AppFooter, { global: { plugins: [createSiteI18n('de')] } })

    const hrefs = screen.getAllByRole('link').map((link) => link.getAttribute('href'))
    expect(hrefs).toEqual(['/de/imprint', '/de/privacy'])
  })
})
