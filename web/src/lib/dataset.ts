import { inject, type InjectionKey } from 'vue'

/** The shapes below mirror internal/content's Dataset and Document, the file
 *  "task content:build" writes to internal/content/generated/dataset.json.
 *  Only the keys a page reads are typed; the Go side reads the file strictly,
 *  this side only needs what it shows. */
export type Lang = 'de' | 'en'

export interface Evidence {
  type: string
  ref: string
}

export interface Skill {
  id: string
  name: string
  level: string
  evidence: Evidence[]
}

export interface Certificate {
  id: string
  name: string
  issuer: string
  issued: string
  valid_until: string | null
}

export interface Experience {
  id: string
  employer: string
  title: string
}

export interface Project {
  id: string
  github_repo: string
  description: string
}

export interface CvDocument {
  language: string
  experience: Experience[]
  certificates: Certificate[]
  skills: Skill[]
  projects: Project[]
}

export interface Dataset {
  /** RFC 3339. The build's clock: pages judge "expired" against it, so a
   *  prerendered page and its hydration agree. */
  generated_at: string
  cv: Record<string, CvDocument>
}

/** datasetKey lets a test or the prerender step hand a page its data. Left
 *  unprovided, pages read the embedded file below. */
export const datasetKey: InjectionKey<Dataset> = Symbol('dataset')

// A glob, not a plain import: the file is gitignored and absent on a clean
// checkout, and a plain import would fail every test that never needs it.
// useDataset fails loudly instead when a page really needs it.
const embedded = import.meta.glob<Dataset>('../../../internal/content/generated/dataset.json', {
  eager: true,
  import: 'default',
})

/** useDataset returns the provided dataset, else the one "task content:build"
 *  wrote. A build without either is a build error, never an empty page. */
export function useDataset(): Dataset {
  const provided = inject(datasetKey, undefined)
  if (provided) {
    return provided
  }
  const [file] = Object.values(embedded)
  if (!file) {
    throw new Error(
      'dataset: internal/content/generated/dataset.json is missing, run task content:build',
    )
  }
  return file
}
