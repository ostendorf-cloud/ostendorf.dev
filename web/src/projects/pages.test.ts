import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

import { createSSRApp, type Component } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createMemoryHistory } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'

import App from '../App.vue'
import { createAppRouter } from '../router'
import { emptyData, fixtureData } from './fixtures'
import { projectLocaleKey, projectsKey } from './data'
import type { ProjectsData } from './types'

/** html server-renders the whole app at path, the way web/scripts/prerender.ts
 *  does and the way a visitor with JavaScript disabled receives it: the
 *  string below is all such a visitor ever gets, no hydration follows. */
async function html(
  path: string,
  data: ProjectsData = fixtureData,
  root: Component = App,
): Promise<string> {
  const router = createAppRouter(createMemoryHistory())
  const app = createSSRApp(root)
  app.use(router)
  app.provide(projectsKey, data)
  await router.push(path)
  await router.isReady()
  return renderToString(app)
}

/** card is the markup of the index entry of one project. */
function card(page: string, name: string): string {
  const entries = page.split('<article').slice(1)
  const found = entries.find((entry) => entry.includes(`>${name}</a>`))
  if (!found) throw new Error(`no card for ${name}`)
  return found.split('</article>')[0]
}

describe('the project index, as served to a visitor with no JavaScript', () => {
  it('lists every project of the dataset, linked to its own page, in id order', async () => {
    const page = await html('/projects/')

    const hrefs = [...page.matchAll(/href="(\/projects\/[a-z-]+\/)"/g)].map((m) => m[1])
    expect(hrefs).toEqual([
      '/projects/alpha-app/',
      '/projects/no-description/',
      '/projects/no-languages/',
      '/projects/no-release/',
      '/projects/zeta-tool/',
    ])
  })

  it('shows the description, the language mix and the latest release', async () => {
    const entry = card(await html('/projects/'), 'alpha-app')

    expect(entry).toContain('An app, the first one.')
    expect(entry).toContain('Go 47%')
    expect(entry).toContain('Vue 47%')
    expect(entry).toContain('Shell 5%')
    expect(entry).toContain('&lt;1%')
    expect(entry).toContain('href="https://github.com/example/alpha-app/releases/tag/v1.2.3"')
    expect(entry).toContain('v1.2.3')
    expect(entry).toContain('2026-10-20')
  })

  it('leaves out the release of a project that has none, without a gap', async () => {
    const entry = card(await html('/projects/'), 'no-release')

    expect(entry).toContain('Not released yet.')
    expect(entry).not.toContain('Latest release')
    expect(entry).not.toContain('<time')
    expect(entry).not.toMatch(/<dd[^>]*>\s*<\/dd>/)
  })

  it('leaves out a missing description, and does not show GitHub text in its place', async () => {
    const entry = card(await html('/projects/'), 'no-description')

    expect(entry).not.toContain('<p')
    expect(entry).not.toContain('GitHub says something.')
    expect(entry).toContain('Latest release')
  })

  it('leaves out the languages of a project with no language data', async () => {
    const entry = card(await html('/projects/'), 'no-languages')

    expect(entry).not.toContain('Languages')
    expect(entry).toContain('A repository of documents.')
  })

  it('writes no placeholder dash anywhere in an entry', async () => {
    const page = await html('/projects/')
    for (const name of ['no-release', 'no-description', 'no-languages']) {
      expect(card(page, name)).not.toMatch(/>\s*[-–—]\s*</)
      expect(card(page, name)).not.toMatch(/\b(n\/a|none|unknown|null|undefined)\b/i)
    }
  })

  it('says when the GitHub data is from, and when that data is stale', async () => {
    expect(await html('/projects/')).toContain('GitHub data as of 2026-11-18.')
    expect(await html('/projects/')).not.toContain('last refresh failed')
    const stale = await html('/projects/', { ...fixtureData, stale: true })
    expect(stale).toContain('last refresh failed')
  })

  it('shows an empty state, not an empty list, when there are no projects', async () => {
    const page = await html('/projects/', emptyData)

    expect(page).toContain('No projects yet')
    expect(page).not.toContain('<article')
  })

  it('carries the heading and marks Projects as the current page', async () => {
    const page = await html('/projects/')

    expect(page).toMatch(/<h1[^>]*>Projects<\/h1>/)
    expect(page).toMatch(/<a[^>]*href="\/projects\/"[^>]*aria-current="page"/)
  })

  it('takes the descriptions of the injected language', async () => {
    const router = createAppRouter(createMemoryHistory())
    const app = createSSRApp(App)
    app.use(router)
    app.provide(projectsKey, fixtureData)
    app.provide(projectLocaleKey, 'de')
    await router.push('/projects/')
    await router.isReady()
    const page = await renderToString(app)

    expect(page).toContain('An app, the first one. (de)')
    expect(page).toContain('Projekte')
  })
})

describe('a project page, as served to a visitor with no JavaScript', () => {
  it('shows the project with its description, facts and link to GitHub', async () => {
    const page = await html('/projects/alpha-app/')

    expect(page).toMatch(/<h1[^>]*>alpha-app<\/h1>/)
    expect(page).toContain('An app, the first one.')
    expect(page).toContain('Go 47%')
    expect(page).toContain('v1.2.3')
    expect(page).toContain('Apache-2.0')
    expect(page).toContain('2026-11-01')
    expect(page).toContain('href="https://github.com/example/alpha-app"')
    expect(page).toContain('href="/projects/"')
  })

  it('has its own page for every project of the dataset', async () => {
    for (const project of fixtureData.projects.en) {
      const page = await html(`/projects/${project.id}/`)
      expect(page).toMatch(new RegExp(`<h1[^>]*>${project.id}</h1>`))
    }
  })

  it('leaves out what the project lacks and marks an archived one', async () => {
    const page = await html('/projects/no-languages/')

    expect(page).not.toContain('Languages')
    expect(page).not.toContain('License')
    expect(page).not.toContain('Topics')
    expect(page).toContain('Archived')
  })

  it('answers an unknown id with its own message, not a blank page', async () => {
    const page = await html('/projects/nope/')

    expect(page).toContain('Project not found')
  })
})

describe('no request leaves the browser', () => {
  it('renders both pages without calling fetch, XMLHttpRequest or sendBeacon', async () => {
    const fetchSpy = vi.fn()
    const xhrSpy = vi.fn()
    const beaconSpy = vi.fn()
    vi.stubGlobal('fetch', fetchSpy)
    vi.stubGlobal('XMLHttpRequest', xhrSpy)
    vi.stubGlobal('WebSocket', xhrSpy)
    vi.stubGlobal('EventSource', xhrSpy)
    Object.defineProperty(navigator, 'sendBeacon', { value: beaconSpy, configurable: true })
    try {
      await html('/projects/')
      await html('/projects/alpha-app/')
    } finally {
      vi.unstubAllGlobals()
    }

    expect(fetchSpy).not.toHaveBeenCalled()
    expect(xhrSpy).not.toHaveBeenCalled()
    expect(beaconSpy).not.toHaveBeenCalled()
  })

  it('has no network call in the source of the project pages', () => {
    const here = import.meta.dirname
    const views = join(here, '../views')
    const files = [
      ...readdirSync(here).map((f) => join(here, f)),
      join(views, 'ProjectsView.vue'),
      join(views, 'ProjectView.vue'),
    ].filter((f) => /\.(ts|vue)$/.test(f) && !f.endsWith('.test.ts'))
    expect(files.length).toBeGreaterThan(5)

    for (const file of files) {
      const source = readFileSync(file, 'utf-8')
      expect(source, file).not.toMatch(
        /\bfetch\s*\(|XMLHttpRequest|sendBeacon|WebSocket|EventSource|createClient|<script[^>]*src=/,
      )
    }
  })
})
