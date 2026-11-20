import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import type { Plugin } from 'vite'

const VIRTUAL_ID = 'virtual:projects-data'
const RESOLVED_ID = '\0' + VIRTUAL_ID

// The merged dataset "task content:build" writes and cmd/website embeds.
const datasetPath = fileURLToPath(
  new URL('../internal/content/generated/dataset.json', import.meta.url),
)

interface RawDataset {
  source: { github: { fetched_at: string; stale: boolean } }
  cv: Record<string, { projects: unknown[] }>
}

/** projectsData exposes the projects of the generated dataset as the module
 *  "virtual:projects-data", and nothing else of it. The dataset also holds the
 *  postal address, the references and the rest of the CV, none of which a page
 *  of this site shows, so none of it may end up in the browser bundle. The
 *  module is null when the dataset has not been built: tests then pass their
 *  own data, and the prerender script fails with a message naming the task. */
export function projectsData(): Plugin {
  return {
    name: 'ostendorf-dev-projects-data',
    resolveId(id) {
      return id === VIRTUAL_ID ? RESOLVED_ID : undefined
    },
    load(id) {
      if (id !== RESOLVED_ID) {
        return undefined
      }
      if (!existsSync(datasetPath)) {
        return 'export default null'
      }
      this.addWatchFile(datasetPath)
      const raw = JSON.parse(readFileSync(datasetPath, 'utf-8')) as RawDataset
      const projects: Record<string, unknown[]> = {}
      for (const [lang, doc] of Object.entries(raw.cv)) {
        projects[lang] = doc.projects
      }
      const data = {
        fetched_at: raw.source.github.fetched_at,
        stale: raw.source.github.stale,
        projects,
      }
      return `export default ${JSON.stringify(data)}`
    },
  }
}
