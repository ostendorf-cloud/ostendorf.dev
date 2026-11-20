/** The shapes of the merged dataset's projects, as internal/content writes
 *  them (Project and Repo in internal/content). Only what a page shows. */

export interface LanguageShare {
  name: string
  bytes: number
}

export interface ReleaseData {
  tag: string
  url: string
  published_at: string
}

export interface RepoData {
  full_name: string
  url: string
  description: string | null
  stars: number
  forks: number
  language: string | null
  languages: LanguageShare[]
  topics: string[]
  license: string | null
  archived: boolean
  pushed_at: string
  latest_release: ReleaseData | null
}

export interface ProjectData {
  id: string
  github_repo: string
  description: string
  github: RepoData
}

export type ProjectLocale = 'de' | 'en'

/** ProjectsData is the part of the dataset the browser bundle carries. */
export interface ProjectsData {
  /** When the GitHub data was fetched, RFC 3339. */
  fetched_at: string
  stale: boolean
  projects: Record<ProjectLocale, ProjectData[]>
}
