import type { ProjectData, ProjectsData, RepoData } from './types'

/** Test data for the project pages: fake repositories, one complete project
 *  and the three that lack a release, a description or language data. The
 *  list is deliberately not in id order. */

function repo(name: string, overrides: Partial<RepoData> = {}): RepoData {
  return {
    full_name: `example/${name}`,
    url: `https://github.com/example/${name}`,
    description: null,
    stars: 5,
    forks: 1,
    language: 'Go',
    languages: [
      { name: 'Shell', bytes: 1000 },
      { name: 'Vue', bytes: 9000 },
      { name: 'Go', bytes: 9000 },
      { name: 'Makefile', bytes: 10 },
    ],
    topics: ['zeta', 'alpha'],
    license: 'Apache-2.0',
    archived: false,
    pushed_at: '2026-11-01T08:30:00Z',
    latest_release: {
      tag: 'v1.2.3',
      url: `https://github.com/example/${name}/releases/tag/v1.2.3`,
      published_at: '2026-10-20T12:00:00Z',
    },
    ...overrides,
  }
}

const enProjects: ProjectData[] = [
  {
    id: 'zeta-tool',
    github_repo: 'example/zeta-tool',
    description: 'A tool that does one thing.',
    github: repo('zeta-tool'),
  },
  {
    id: 'alpha-app',
    github_repo: 'example/alpha-app',
    description: 'An app, the first one.',
    github: repo('alpha-app'),
  },
  {
    id: 'no-release',
    github_repo: 'example/no-release',
    description: 'Not released yet.',
    github: repo('no-release', { latest_release: null }),
  },
  {
    id: 'no-description',
    github_repo: 'example/no-description',
    description: '  ',
    github: repo('no-description', { description: 'GitHub says something.' }),
  },
  {
    id: 'no-languages',
    github_repo: 'example/no-languages',
    description: 'A repository of documents.',
    github: repo('no-languages', {
      language: null,
      languages: [],
      topics: [],
      license: null,
      archived: true,
    }),
  },
]

function inGerman(projects: ProjectData[]): ProjectData[] {
  return projects.map((p) => ({
    ...p,
    description: p.description.trim() === '' ? p.description : `${p.description} (de)`,
  }))
}

export const fixtureData: ProjectsData = {
  fetched_at: '2026-11-18T10:00:00Z',
  stale: false,
  projects: { en: enProjects, de: inGerman(enProjects) },
}

export const emptyData: ProjectsData = {
  fetched_at: '2026-11-18T10:00:00Z',
  stale: false,
  projects: { en: [], de: [] },
}
