import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { useTheme } from '@ostendorf-cloud/govue'
import { beforeEach, describe, expect, it } from 'vitest'

import ThemeToggle from './ThemeToggle.vue'

describe('ThemeToggle', () => {
  // useTheme's state is a module level singleton, by design: one page has
  // one theme. A test that leaves mode changed would leak into the next
  // one, so every test starts by forcing it back to "system" through the
  // same public setMode the component itself calls.
  beforeEach(() => {
    useTheme().setMode('system')
  })

  it('names light, the first step of the cycle, as the action at the start', () => {
    render(ThemeToggle)

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeVisible()
  })

  it('cycles through light, dark and back to system on repeated clicks', async () => {
    const user = userEvent.setup()
    render(ThemeToggle)

    const button = screen.getByRole('button')
    await user.click(button)
    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeVisible()

    await user.click(button)
    expect(screen.getByRole('button', { name: 'Follow system theme' })).toBeVisible()

    await user.click(button)
    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeVisible()
  })
})
