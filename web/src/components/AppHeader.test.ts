import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import { createAppRouter } from '../router'
import AppHeader from './AppHeader.vue'

async function renderHeader() {
  const router = createAppRouter(createMemoryHistory())
  await router.push({ name: 'home' })
  await router.isReady()
  return render(AppHeader, { global: { plugins: [router] } })
}

describe('AppHeader', () => {
  it('names the site in the brand link', async () => {
    await renderHeader()

    expect(screen.getByRole('link', { name: 'Simon Ostendorf' })).toHaveAttribute('href', '/')
  })

  it('marks the home page as the current page in the main navigation', async () => {
    await renderHeader()

    expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')
  })

  it('carries the theme toggle', async () => {
    await renderHeader()

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeVisible()
  })
})
