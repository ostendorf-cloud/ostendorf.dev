import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from '../App.vue'
import { createAppRouter } from '../router'
import { createSiteI18n, followRoute, localeOfPath } from './language'

async function renderAt(path: string) {
  const i18n = createSiteI18n(localeOfPath(path))
  const router = createAppRouter(createMemoryHistory())
  followRoute(router, i18n)
  await router.push(path)
  await router.isReady()
  render(App, { global: { plugins: [i18n, router] } })
  return { i18n, router }
}

function footerHrefs(): (string | null)[] {
  return [...screen.getByRole('contentinfo').querySelectorAll('a')].map((a) =>
    a.getAttribute('href'),
  )
}

describe('localeOfPath', () => {
  it('is the prefix language, and English everywhere else', () => {
    expect(localeOfPath('/de/privacy')).toBe('de')
    expect(localeOfPath('/en/imprint')).toBe('en')
    expect(localeOfPath('/')).toBe('en')
    expect(localeOfPath('/nowhere/')).toBe('en')
  })
})

describe('the language follows the address', () => {
  it('returns to English when Back goes from a German page to the English one', async () => {
    const { i18n, router } = await renderAt('/en/imprint')

    // What a language switch does before the full page load: set the
    // language, then go to the other page.
    i18n.language.setLocale('de', { persist: false })
    await router.push('/de/imprint')
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Impressum'),
    )

    router.back()
    await waitFor(() => expect(router.currentRoute.value.path).toBe('/en/imprint'))

    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Imprint'),
    )
    expect(i18n.language.locale.value).toBe('en')
    expect(footerHrefs()).toEqual(['/en/imprint', '/en/privacy'])
  })

  it('shows the English home page with English footer links from the brand link of a German page', async () => {
    const { i18n } = await renderAt('/de/privacy')
    expect(footerHrefs()).toEqual(['/de/imprint', '/de/privacy'])

    await userEvent.click(screen.getByRole('banner').querySelector('a') as HTMLElement)

    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Simon Ostendorf'),
    )
    expect(i18n.language.locale.value).toBe('en')
    expect(footerHrefs()).toEqual(['/en/imprint', '/en/privacy'])
  })

  it('keeps German when a German page is entered directly', async () => {
    const { i18n } = await renderAt('/de/privacy')

    expect(i18n.language.locale.value).toBe('de')
  })
})

describe('the language is never stored', () => {
  it('writes no cookie and no local storage entry when the language changes', async () => {
    document.cookie.split(';').forEach((c) => {
      document.cookie = `${c.split('=')[0]}=; Max-Age=0; Path=/`
    })
    localStorage.clear()
    const { i18n, router } = await renderAt('/en/imprint')

    // persist defaults to true: the storage option alone must stop the write.
    i18n.language.setLocale('de')
    await router.push('/de/imprint')

    expect(document.cookie).toBe('')
    expect(localStorage.length).toBe(0)
  })
})
