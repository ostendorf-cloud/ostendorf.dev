import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import AppFooter from './AppFooter.vue'

describe('AppFooter', () => {
  it('carries a copyright note for the current year', () => {
    render(AppFooter)

    expect(screen.getByText(`© ${new Date().getFullYear()} Simon Ostendorf`)).toBeVisible()
  })

  // See AGENTS.md: simon-ostendorf.de is the hub that links out to the
  // other two sites, so this footer deliberately carries none of its
  // own.
  it('carries no link to a sibling site', () => {
    render(AppFooter)

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
