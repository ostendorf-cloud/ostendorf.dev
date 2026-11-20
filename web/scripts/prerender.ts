/**
 * Renders every route to static HTML, so a crawler and a browser's first
 * paint both get real markup instead of an empty <div id="app">. Run
 * through "task prerender", after "task web:build" has written the SPA
 * bundle and its manifest into cmd/website/dist.
 *
 * The Go service embeds the files this script writes into
 * cmd/website/prerendered and serves them directly; the SPA bundle this
 * same page loads then hydrates over the server-rendered markup rather
 * than replacing it, through createSSRApp on both sides.
 */
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createMemoryHistory, type RouteLocationRaw } from 'vue-router'
import { useTheme } from '@ostendorf-cloud/govue'

import App from '../src/App.vue'
import { routeTable } from '../src/lib/routeTable'
import {
  canonical,
  fullTitle,
  NOT_FOUND,
  pageMeta,
  PROJECT,
  projectCanonical,
  projectMeta,
  type PageMeta,
  type PageRoute,
} from '../src/lib/seo'
import { THEME_NAME } from '../src/lib/theme'
import { embeddedProjects } from '../src/projects/data'
import { assertProjectIDs, projectList } from '../src/projects/model'
import { createAppRouter, NOT_FOUND_ROUTE_NAME, PROJECT_ROUTE_NAME } from '../src/router'

const here = dirname(fileURLToPath(import.meta.url))
const distDir = join(here, '../../cmd/website/dist')
const outDir = join(here, '../../cmd/website/prerendered')

// The same call main.ts makes, before anything is rendered: see that
// file for why. There is no document here, so the attribute writing
// half of useTheme does nothing; renderDocument below writes
// data-theme itself.
useTheme({ theme: THEME_NAME })

/** loadPages turns the shared route table into this script's own page
 *  list, pairing each entry's name with the router route and the SEO
 *  route it names, plus the one page the table itself does not carry:
 *  the 404 page, which answers no route and so is not part of the table
 *  internal/webapp's sitemap and mux registration read either. */
function loadPages(): Page[] {
  const pages: Page[] = routeTable.map((entry) => ({
    location: { name: entry.name },
    seoRoute: entry.name,
    meta: pageMeta(entry.name),
    url: canonical(entry.name),
    file: entry.file,
  }))
  pages.push({
    location: { name: NOT_FOUND_ROUTE_NAME },
    seoRoute: NOT_FOUND,
    meta: pageMeta(NOT_FOUND),
    url: canonical(NOT_FOUND),
    file: '404.html',
  })
  pages.push(...projectPages())
  return pages
}

/** projectPages is one page per project of the generated dataset, written to
 *  projects/<id>/index.html: the paths internal/webapp serves from the same
 *  dataset. The site is English only for now, so the English texts are used. */
function projectPages(): Page[] {
  const projects = projectList(embeddedProjects(), 'en')
  assertProjectIDs(projects)
  return projects.map((project) => ({
    location: { name: PROJECT_ROUTE_NAME, params: { id: project.id } },
    seoRoute: PROJECT,
    meta: projectMeta(project),
    url: projectCanonical(project.id),
    file: `projects/${project.id}/index.html`,
  }))
}

interface Page {
  location: RouteLocationRaw
  /** The route name fullTitle needs: only "home" gets no suffix. */
  seoRoute: PageRoute
  meta: PageMeta
  url: string
  file: string
}

interface ManifestEntry {
  file: string
  css?: string[]
}

interface Assets {
  script: string
  styles: string[]
}

/** stylesheets is every stylesheet the build wrote, whatever shape the
 *  manifest records it in: as a chunk's own "css" list, or as a
 *  standalone entry when the build does not split CSS at all.
 *
 *  All of them, not the entry's alone. A prerendered page is served as
 *  static HTML and the module graph that would pull a route chunk's own
 *  stylesheet in runs after first paint, or never at all for a visitor
 *  with no JavaScript. Linking only the entry's CSS meant a view with a
 *  <style> block of its own painted unstyled until then; linking all of
 *  them is correct for any view added later and for either setting of
 *  build.cssCodeSplit, rather than only for the one this repository
 *  happens to use. */
function stylesheets(manifest: Record<string, ManifestEntry>): string[] {
  const entry = manifest['index.html']
  const found = new Set<string>(entry?.css ?? [])
  for (const record of Object.values(manifest)) {
    for (const file of record.css ?? []) {
      found.add(file)
    }
    if (record.file.endsWith('.css')) {
      found.add(record.file)
    }
  }
  if (found.size === 0) {
    throw new Error('prerender: the vite manifest records no stylesheet, run task web:build first')
  }
  return [...found]
}

async function loadManifest(): Promise<Assets> {
  const raw = await readFile(join(distDir, '.vite/manifest.json'), 'utf-8')
  const manifest = JSON.parse(raw) as Record<string, ManifestEntry>
  // The manifest keys its one HTML entry by the input file vite built,
  // index.html, not by the script tag inside it: that script's own
  // bundle is this entry's "file".
  const entry = manifest['index.html']
  if (!entry) {
    throw new Error('prerender: no index.html entry in the vite manifest, run task web:build first')
  }
  return { script: entry.file, styles: stylesheets(manifest) }
}

/** renderBody server-renders the app at routerName, with a fresh router
 *  instance per page: nothing from one page's render may leak into the
 *  next. */
async function renderBody(location: RouteLocationRaw): Promise<string> {
  const router = createAppRouter(createMemoryHistory())

  const app = createSSRApp(App)
  app.use(router)

  await router.push(location)
  await router.isReady()

  return renderToString(app)
}

function renderHead(page: Page, assets: Assets): string {
  const meta = page.meta
  const title = fullTitle(page.seoRoute, meta)
  const url = page.url

  const styleLinks = assets.styles
    .map((href) => `<link rel="stylesheet" href="/${href}">`)
    .join('\n    ')

  return `<meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>${escapeHtml(title)}</title>
    <meta name="description" content="${escapeHtml(meta.description)}">
    <meta name="robots" content="index, follow">
    <link rel="canonical" href="${url}">
    <meta property="og:type" content="website">
    <meta property="og:site_name" content="Simon Ostendorf">
    <meta property="og:title" content="${escapeHtml(title)}">
    <meta property="og:description" content="${escapeHtml(meta.description)}">
    <meta property="og:url" content="${url}">
    ${styleLinks}
    <script type="module" src="/${assets.script}" defer></script>`
}

/** renderDocument wraps one rendered page. data-theme is the one
 *  attribute govue's stylesheets switch the palette on, written here
 *  rather than left to useTheme so the very first paint of a prerendered
 *  page already carries it. There is no data-mode attribute: this site's
 *  mode stays at useTheme's own "system" default until a visitor's own
 *  choice through the header's theme toggle overrides it, and an absent
 *  data-mode is exactly how useTheme spells "follow the operating
 *  system" (see useTheme's own apply()), so pinning one here would be
 *  wrong for a visitor whose system disagrees with it. */
function renderDocument(head: string, body: string): string {
  return `<!doctype html>
<html lang="en" data-theme="${THEME_NAME}">
  <head>
    ${head}
  </head>
  <body>
    <div id="app">${body}</div>
  </body>
</html>
`
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

async function main(): Promise<void> {
  const assets = await loadManifest()
  const pages = loadPages()

  for (const page of pages) {
    const body = await renderBody(page.location)
    const head = renderHead(page, assets)
    const html = renderDocument(head, body)

    const target = join(outDir, page.file)
    await mkdir(dirname(target), { recursive: true })
    await writeFile(target, html, 'utf-8')
    console.log(`wrote ${target}`)
  }
}

main().catch((error: unknown) => {
  console.error(error)
  process.exitCode = 1
})
