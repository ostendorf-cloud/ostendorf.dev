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

  it('states own hardware and names no hosting provider, without the old hedging', async () => {
    for (const [locale, own, gone] of [
      [
        'de',
        'eigener Hardware',
        ['Hetzner', 'wenn sie für die Website eingerichtet ist', 'kann Seitenaufrufe'],
      ],
      ['en', 'own hardware', ['Hetzner', 'when it is set up for the site', 'can count page views']],
    ] as const) {
      const { container, unmount } = await renderAt(`/${locale}/privacy`, locale)
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')
      expect(text).toContain(own)
      for (const phrase of gone) {
        expect(text).not.toContain(phrase)
      }
      unmount()
    }
  })

  it.each([
    ['de', ['nach 30 Tagen', 'nach 14 Monaten', 'Art. 6 Abs. 1 lit. f DSGVO']],
    ['en', ['after 30 days', 'after 14 months', 'Art. 6(1)(f) GDPR']],
  ] as const)('states retention and legal basis in %s', async (locale, needles) => {
    const { container } = await renderAt(`/${locale}/privacy`, locale)
    const text = (container.textContent ?? '').replace(/\s+/g, ' ')
    for (const needle of needles) {
      expect(text).toContain(needle)
    }
  })

  it.each([
    [
      'de',
      [
        'Unser berechtigtes Interesse ist der sichere Betrieb der Website, die Analyse von Fehlern und die Abwehr von Missbrauch.',
        'Unser berechtigtes Interesse ist, zu verstehen, wie die Website genutzt wird.',
      ],
      'Wir haben dieses Interesse gegen Ihre Rechte abgewogen und halten die Daten gering.',
    ],
    [
      'en',
      [
        'Our legitimate interest is the secure operation of the site, the analysis of errors and the defence against abuse.',
        'Our legitimate interest is to understand how the site is used.',
      ],
      'We weighed this interest against your rights and keep the data minimal.',
    ],
  ] as const)(
    'states one interest per purpose and the weighing in %s',
    async (locale, purposes, weighing) => {
      const { container } = await renderAt(`/${locale}/privacy`, locale)
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')
      for (const purpose of purposes) {
        expect(text).toContain(purpose)
      }
      expect(text.split(weighing)).toHaveLength(3)
    },
  )

  it.each([
    ['de', 'eigener Hardware in Deutschland', ['zu Hause', 'Hetzner', 'anonyme']],
    ['en', 'own hardware in Germany', ['at home', 'Hetzner', 'anonymous']],
  ] as const)('states the location as %s and nothing else', async (locale, own, gone) => {
    const { container } = await renderAt(`/${locale}/privacy`, locale)
    const text = (container.textContent ?? '').replace(/\s+/g, ' ')
    expect(text).toContain(own)
    for (const phrase of gone) {
      expect(text).not.toContain(phrase)
    }
  })

  it.each([
    [
      'de',
      'Das Protokoll der Anwendung enthält pro Aufruf:',
      [
        'den Zeitpunkt des Aufrufs,',
        'die Anfragemethode, zum Beispiel GET,',
        'die Seite als Routenmuster, nicht die aufgerufene Adresse mit ihren Parametern,',
        'den HTTP-Statuscode und die übertragene Datenmenge,',
        'die Bearbeitungsdauer,',
        'Ihre IP-Adresse, ermittelt hinter den vorgeschalteten Proxys.',
      ],
      /user-agent|referrer|verweisende/i,
      'Das Protokoll des vorgeschalteten Ingress, der Ihre Anfrage zuerst annimmt, enthält in der üblichen Form eines Webserver-Protokolls:',
      [
        'Ihre IP-Adresse,',
        'Datum und Uhrzeit der Anfrage,',
        'die Anfragezeile (aufgerufene Seite bzw. Datei mit Parametern, Anfragemethode und Protokollversion),',
        'den HTTP-Statuscode und die übertragene Datenmenge,',
        'den User-Agent Ihres Browsers,',
        'die verweisende Adresse (Referrer), sofern Ihr Browser sie übermittelt.',
      ],
    ],
    [
      'en',
      'The log of the application contains, per request:',
      [
        'the time of the request,',
        'the request method, for example GET,',
        'the page as a route pattern, not the requested address with its parameters,',
        'the HTTP status code and the amount of data sent,',
        'the processing time,',
        'your IP address, determined behind the proxies in front.',
      ],
      /user agent|referrer|referring/i,
      'The log of the ingress in front of the application, which accepts your request first, contains, in the usual form of a web server log:',
      [
        'your IP address,',
        'the date and time of the request,',
        'the request line (the page or file requested with its parameters, the request method and the protocol version),',
        'the HTTP status code and the amount of data sent,',
        'the user agent of your browser,',
        'the referring address (referrer), where your browser sends it.',
      ],
    ],
  ] as const)(
    'lists the exact fields of the application log and the ingress log in %s',
    async (locale, appIntro, appFields, notLogged, ingressIntro, ingressFields) => {
      const { container } = await renderAt(`/${locale}/privacy`, locale)
      const listAfter = (intro: string) => {
        const p = [...container.querySelectorAll('p')].find(
          (el) => (el.textContent ?? '').replace(/\s+/g, ' ').trim() === intro,
        )
        expect(p, intro).toBeDefined()
        const ul = p?.nextElementSibling
        expect(ul?.tagName).toBe('UL')
        return [...(ul?.querySelectorAll('li') ?? [])].map((li) =>
          (li.textContent ?? '').replace(/\s+/g, ' ').trim(),
        )
      }

      const app = listAfter(appIntro)
      expect(app).toEqual(appFields)
      expect(app.join(' ')).not.toMatch(notLogged)
      expect(listAfter(ingressIntro)).toEqual(ingressFields)
    },
  )

  it.each([
    [
      'de',
      [
        'Protokoll der Anwendung',
        'Routenmuster',
        'User-Agent und die verweisende Adresse (Referrer) stehen nicht im Protokoll der Anwendung',
        'Protokoll des vorgeschalteten Ingress',
        'Anfragezeile',
        'Beide Protokolle werden nach 30 Tagen gelöscht',
      ],
    ],
    [
      'en',
      [
        'log of the application',
        'route pattern',
        'user agent and the referring address (referrer) are not in the application log',
        'log of the ingress',
        'request line',
        'Both logs are deleted after 30 days',
      ],
    ],
  ] as const)(
    'states the application log and the ingress log apart in %s',
    async (locale, needles) => {
      const { container } = await renderAt(`/${locale}/privacy`, locale)
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')
      for (const needle of needles) {
        expect(text).toContain(needle)
      }
    },
  )

  it.each([
    ['de', 'Protokoll, Domain und gegebenenfalls Port', 'pseudonyme Kennung', 'Art. 21 DSGVO'],
    ['en', 'protocol, domain and, where present, port', 'pseudonymous identifier', 'Art. 21 GDPR'],
  ] as const)(
    'states the referrer port, the pseudonym and the objection in %s',
    async (locale, ...needles) => {
      const { container } = await renderAt(`/${locale}/privacy`, locale)
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')
      for (const needle of needles) {
        expect(text).toContain(needle)
      }
    },
  )

  it.each([
    ['de', 'Die Kennung wechselt jeden Monat.', 'deshalb'],
    ['en', 'The identifier changes every month.', 'therefore'],
  ] as const)('states the monthly identifier plainly in %s', async (locale, stated, gone) => {
    const { container } = await renderAt(`/${locale}/privacy`, locale)
    const text = (container.textContent ?? '').replace(/\s+/g, ' ')
    expect(text).toContain(stated)
    expect(text).not.toContain(gone)
  })

  it.each([
    ['de', 'Wenn Sie mir', 'erhalte ich'],
    ['en', 'write me', 'I receive'],
  ] as const)('speaks in one voice in %s', async (locale, ...gone) => {
    const { container } = await renderAt(`/${locale}/privacy`, locale)
    const text = (container.textContent ?? '').replace(/\s+/g, ' ')
    for (const phrase of gone) {
      expect(text).not.toContain(phrase)
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
