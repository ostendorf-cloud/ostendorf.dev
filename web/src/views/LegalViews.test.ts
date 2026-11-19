import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from '../App.vue'
import { createSiteI18n } from '../lib/language'
import { createAppRouter } from '../router'

async function renderAt(path: string, locale: 'de' | 'en') {
  const router = createAppRouter(createMemoryHistory())
  await router.push(path)
  await router.isReady()
  return render(App, { global: { plugins: [createSiteI18n(locale), router] } })
}

const pages = [
  { path: '/de/imprint', locale: 'de', h1: 'Impressum' },
  { path: '/en/imprint', locale: 'en', h1: 'Imprint' },
  { path: '/de/privacy', locale: 'de', h1: 'Datenschutzerklärung' },
  { path: '/en/privacy', locale: 'en', h1: 'Privacy policy' },
] as const

describe('legal pages', () => {
  it.each(pages)('renders $path in $locale with its own heading', async (page) => {
    await renderAt(page.path, page.locale)

    expect(screen.getByRole('heading', { level: 1 }).textContent).toContain(page.h1)
  })

  it.each(pages)(
    'links the imprint and the privacy page of its language from $path',
    async (page) => {
      await renderAt(page.path, page.locale)

      const footer = screen.getByRole('contentinfo')
      const hrefs = [...footer.querySelectorAll('a')].map((a) => a.getAttribute('href'))
      expect(hrefs).toContain(`/${page.locale}/imprint`)
      expect(hrefs).toContain(`/${page.locale}/privacy`)
    },
  )

  it.each(['de', 'en'] as const)('states the operator in the %s imprint', async (locale) => {
    const { container } = await renderAt(`/${locale}/imprint`, locale)
    const text = container.textContent ?? ''

    for (const fact of [
      'Simon Ostendorf',
      'Frankenstr. 8',
      '52070 Aachen',
      '+49 1573 0971784',
      'simon@ostendorf.dev',
      'DE362164357',
    ]) {
      expect(text).toContain(fact)
    }
  })

  it.each(pages)('has no dispute resolution or contact form text on $path', async (page) => {
    const { container } = await renderAt(page.path, page.locale)
    const text = (container.textContent ?? '').toLowerCase()

    for (const banned of [
      'verbraucherstreit',
      'streitbeilegung',
      'schlichtung',
      'dispute resolution',
      'arbitration',
      'kontaktformular',
      'contact form',
    ]) {
      expect(text).not.toContain(banned)
    }
  })

  it('states the page view counting fields in both languages', async () => {
    for (const [locale, needles] of [
      [
        'de',
        ['Umami', 'verkürzt', 'kein Cookie', 'Land, Region und Stadt', 'Do Not Track', 'Sec-GPC'],
      ],
      [
        'en',
        ['Umami', 'shortened', 'no cookie', 'country, region and city', 'Do Not Track', 'Sec-GPC'],
      ],
    ] as const) {
      const { container, unmount } = await renderAt(`/${locale}/privacy`, locale)
      for (const needle of needles) {
        expect(container.textContent).toContain(needle)
      }
      unmount()
    }
  })

  it('links the German version from the English pages', async () => {
    await renderAt('/en/privacy', 'en')

    expect(screen.getByRole('link', { name: 'View the German version' })).toHaveAttribute(
      'href',
      '/de/privacy',
    )
  })
})
