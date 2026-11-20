import { render, screen, within } from '@testing-library/vue'
import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from '../App.vue'
import { datasetKey } from '../lib/dataset'
import { localeKey } from '../i18n/skills'
import { fixtureDataset } from '../lib/skills.fixture'
import { canonical, fullTitle, pageMeta } from '../lib/seo'
import { createAppRouter } from '../router'
import SkillsView from './SkillsView.vue'

function renderView(lang: 'en' | 'de' = 'en') {
  return render(SkillsView, {
    global: {
      provide: { [datasetKey as symbol]: fixtureDataset, [localeKey as symbol]: lang },
    },
  })
}

function skillItem(name: string): HTMLElement {
  const item = screen.getByRole('heading', { level: 3, name }).closest('li')
  if (!item) {
    throw new Error(`no list item around ${name}`)
  }
  return item
}

describe('SkillsView', () => {
  it('groups the skills under the level in words, highest first', () => {
    renderView()

    const levels = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
    expect(levels).toEqual(['Expert', 'Proficient', 'Working knowledge', 'Familiar'])
    expect(screen.queryByRole('progressbar')).toBeNull()
    expect(document.body.textContent).not.toMatch(/%/)
  })

  it('links project evidence to the project page', () => {
    renderView()

    expect(
      within(skillItem('Kubernetes en')).getByRole('link', { name: 'example/repo' }),
    ).toHaveAttribute('href', '/projects/#proj-1')
  })

  it('shows the issuer of a certificate and marks an expired one', () => {
    renderView()

    const item = skillItem('Kubernetes en')
    expect(item).toHaveTextContent('Old Cert en')
    expect(item).toHaveTextContent('Issuer: Old Body en')
    expect(within(item).getAllByText('Expired')).toHaveLength(1)
    expect(item).toHaveTextContent('Issuer: New Body en, valid until 2027-12')
  })

  it('shows a certificate without an end date as not expired', () => {
    renderView()

    const item = skillItem('OpenTofu en')
    expect(item).toHaveTextContent('Issuer: Forever Body en')
    expect(within(item).queryByText('Expired')).toBeNull()
  })

  it('shows a position as plain text', () => {
    renderView()

    const item = skillItem('Go en')
    expect(item).toHaveTextContent('Position: Engineer en, Employer en')
    expect(within(item).queryByRole('link')).toBeNull()
  })

  it('does not imply evidence for a skill without any', () => {
    renderView()

    const item = skillItem('Rust en')
    expect(item).toHaveTextContent('No evidence listed.')
    expect(within(item).queryByText('Evidence')).toBeNull()
    expect(within(item).queryByRole('list')).toBeNull()
  })

  it('keeps a reference that points at nothing instead of dropping it', () => {
    renderView()

    expect(skillItem('Dangling en')).toHaveTextContent('proj-missing')
  })

  it('speaks German from the German document', () => {
    renderView('de')

    expect(screen.getByRole('heading', { level: 1, name: 'Skills und Werkzeuge' })).toBeVisible()
    expect(screen.getByRole('heading', { level: 2, name: 'Experte' })).toBeVisible()
    const item = skillItem('Kubernetes de')
    expect(item).toHaveTextContent('Aussteller: Old Body de')
    expect(within(item).getByText('Abgelaufen')).toBeVisible()
    expect(skillItem('Rust de')).toHaveTextContent('Keine Belege angegeben.')
  })
})

// The prerender step renders the app the same way, with renderToString on a
// memory history, and writes the string into the page. With no JavaScript the
// visitor reads exactly this.
describe('the prerendered skills page', () => {
  async function prerender(): Promise<string> {
    const router = createAppRouter(createMemoryHistory())
    const app = createSSRApp(App)
    app.use(router)
    app.provide(datasetKey, fixtureDataset)
    await router.push({ name: 'skills' })
    await router.isReady()
    return renderToString(app)
  }

  it('carries the skills, their levels and their evidence as static HTML', async () => {
    const html = await prerender()

    expect(html).toContain('Skills and tools')
    expect(html).toContain('Kubernetes en')
    expect(html).toContain('Expert')
    expect(html).toContain('href="/projects/#proj-1"')
    expect(html).toContain('Issuer: Old Body en')
    expect(html).toContain('Expired')
    expect(html).toContain('No evidence listed.')
    expect(html).not.toContain('<script')
  })

  it('has its own title and description from the shared SEO helper', () => {
    const meta = pageMeta('skills')
    const home = pageMeta('home')

    expect(meta.title).toBe('Skills and tools')
    expect(fullTitle('skills', meta)).toBe('Skills and tools – Simon Ostendorf')
    expect(meta.description).not.toBe(home.description)
    expect(meta.description).toContain('evidence')
    expect(canonical('skills')).toMatch(/\/skills\/$/)
  })
})
