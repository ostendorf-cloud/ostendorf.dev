import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import App from './App.vue'

describe('App', () => {
  it('renders the one placeholder page with its heading', () => {
    render(App)

    expect(screen.getByRole('heading', { level: 1, name: 'Simon Ostendorf' })).toBeVisible()
  })

  it('carries the real profile and contact links from the predecessor site', () => {
    render(App)

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
      'mailto:simon.ostendorf@stackmeister.com',
    )
  })

  it('has a skip link targeting the main landmark', () => {
    render(App)

    expect(screen.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main')
  })
})
