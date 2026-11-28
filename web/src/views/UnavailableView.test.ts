import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import { createAppRouter } from '../router'
import UnavailableView from './UnavailableView.vue'

describe('UnavailableView', () => {
  it('names the page as temporarily unavailable', async () => {
    const router = createAppRouter(createMemoryHistory())
    await router.push({ name: 'unavailable' })
    await router.isReady()

    render(UnavailableView, { global: { plugins: [router] } })

    expect(screen.getByRole('heading', { level: 1, name: 'Temporarily unavailable' })).toBeVisible()
  })

  it('links back to the home page', async () => {
    const router = createAppRouter(createMemoryHistory())
    await router.push({ name: 'unavailable' })
    await router.isReady()

    render(UnavailableView, { global: { plugins: [router] } })

    expect(screen.getByRole('link', { name: 'Back to the home page' })).toHaveAttribute('href', '/')
  })
})
