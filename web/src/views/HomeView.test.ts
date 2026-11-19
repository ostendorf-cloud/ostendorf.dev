import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import HomeView from './HomeView.vue'

describe('HomeView', () => {
  it('renders the one placeholder page with its heading', () => {
    render(HomeView)

    expect(screen.getByRole('heading', { level: 1, name: 'Simon Ostendorf' })).toBeVisible()
  })

  it('carries the real profile and contact links from the predecessor site', () => {
    render(HomeView)

    expect(screen.getByRole('link', { name: 'GitHub' })).toHaveAttribute(
      'href',
      'https://github.com/simonostendorf',
    )
    expect(screen.getByRole('link', { name: 'LinkedIn' })).toHaveAttribute(
      'href',
      'https://www.linkedin.com/in/simonostendorf',
    )
    expect(screen.getByRole('link', { name: 'Email' })).toHaveAttribute(
      'href',
      'mailto:simon@ostendorf.dev',
    )
  })
})
