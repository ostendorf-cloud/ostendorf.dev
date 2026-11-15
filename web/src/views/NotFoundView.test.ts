import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import { createAppRouter } from '../router'
import NotFoundView from './NotFoundView.vue'

describe('NotFoundView', () => {
  it('names the page as not found', async () => {
    const router = createAppRouter(createMemoryHistory())
    await router.push('/this-page-does-not-exist/')
    await router.isReady()

    render(NotFoundView, { global: { plugins: [router] } })

    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeVisible()
  })

  it('links back to the home page', async () => {
    const router = createAppRouter(createMemoryHistory())
    await router.push('/this-page-does-not-exist/')
    await router.isReady()

    render(NotFoundView, { global: { plugins: [router] } })

    expect(screen.getByRole('link', { name: 'Back to the home page' })).toHaveAttribute('href', '/')
  })
})
