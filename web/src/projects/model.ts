import type { ProjectData, ProjectLocale, ProjectsData } from './types'

/** PROJECT_ID_PATTERN is what a project id must look like to be a path
 *  segment. internal/webapp/projects.go holds the same rule for the server. */
export const PROJECT_ID_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/

export interface LanguageShare {
  name: string
  /** Text such as "47%", or "<1%" for a share below one percent. */
  share: string
}

export interface ReleaseInfo {
  tag: string
  url: string
  /** The publication date as YYYY-MM-DD. */
  date: string
}

/** Project is what a page shows of one project. A part that is absent is
 *  null or an empty list, never a placeholder: the view leaves it out. */
export interface Project {
  id: string
  name: string
  fullName: string
  /** The hand written text from the cv repository, null when it is empty. */
  description: string | null
  githubUrl: string
  languages: LanguageShare[]
  release: ReleaseInfo | null
  stars: number
  forks: number
  topics: string[]
  license: string | null
  archived: boolean
  pushedAt: string | null
}

/** languageMix turns byte counts into shares, largest first and by name on a
 *  tie, so the same data always gives the same list. */
export function languageMix(
  languages: readonly { name: string; bytes: number }[],
): LanguageShare[] {
  const counted = languages.filter((l) => l.bytes > 0)
  const total = counted.reduce((sum, l) => sum + l.bytes, 0)
  return [...counted]
    .sort((a, b) => b.bytes - a.bytes || a.name.localeCompare(b.name, 'en'))
    .map((l) => {
      const percent = (l.bytes / total) * 100
      return { name: l.name, share: percent < 1 ? '<1%' : `${Math.round(percent)}%` }
    })
}

function dateOf(timestamp: string): string | null {
  const match = /^\d{4}-\d{2}-\d{2}/.exec(timestamp)
  return match ? match[0] : null
}

/** toProject maps one dataset project to the page model. */
export function toProject(data: ProjectData): Project {
  const repo = data.github
  const description = data.description.trim()
  const published = repo.latest_release ? dateOf(repo.latest_release.published_at) : null
  return {
    id: data.id,
    name: repo.full_name.split('/').pop() ?? repo.full_name,
    fullName: repo.full_name,
    description: description === '' ? null : description,
    githubUrl: repo.url,
    languages: languageMix(repo.languages),
    release:
      repo.latest_release && published
        ? { tag: repo.latest_release.tag, url: repo.latest_release.url, date: published }
        : null,
    stars: repo.stars,
    forks: repo.forks,
    topics: [...repo.topics].sort(),
    license: repo.license && repo.license !== 'NOASSERTION' ? repo.license : null,
    archived: repo.archived,
    pushedAt: dateOf(repo.pushed_at),
  }
}

/** projectList is every project of one language in a fixed order: by id. The
 *  order of the cv file is not relied on, so an unchanged set of projects
 *  renders the same list in every build. */
export function projectList(data: ProjectsData, locale: ProjectLocale): Project[] {
  return data.projects[locale]
    .map(toProject)
    .sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
}

/** fetchedDate is the day the GitHub data was fetched, YYYY-MM-DD. */
export function fetchedDate(data: ProjectsData): string | null {
  return dateOf(data.fetched_at)
}

/** assertProjectIDs fails on an id that is not a path segment or that
 *  repeats, so the prerender script cannot write an unreachable page. */
export function assertProjectIDs(projects: readonly { id: string }[]): void {
  const seen = new Set<string>()
  for (const { id } of projects) {
    if (!PROJECT_ID_PATTERN.test(id)) {
      throw new Error(`project id "${id}" is not a path segment: use lower case words and hyphens`)
    }
    if (seen.has(id)) {
      throw new Error(`project id "${id}" appears twice`)
    }
    seen.add(id)
  }
}
