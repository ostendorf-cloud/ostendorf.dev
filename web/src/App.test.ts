import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from './App.vue'
import { createAppRouter } from './router'

async function renderApp(path = '/') {
  const router = createAppRouter(createMemoryHistory())
  await router.push(path)
  await router.isReady()
  return render(App, { global: { plugins: [router] } })
}

describe('App', () => {
  it('has a skip link targeting the main landmark', async () => {
    await renderApp()

    expect(screen.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main')
  })

  it('carries the header and the footer around whichever page the router resolves', async () => {
    await renderApp()

    expect(screen.getByRole('banner')).toBeVisible()
    expect(screen.getByRole('contentinfo')).toBeVisible()
  })

  it('renders the home page at the root path', async () => {
    await renderApp('/')

    expect(screen.getByRole('heading', { level: 1, name: 'Simon Ostendorf' })).toBeVisible()
  })

  it('renders the 404 page for a path the route table does not know', async () => {
    await renderApp('/this-page-does-not-exist/')

    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeVisible()
  })
})
