import { describe, expect, it } from 'vitest'

import { fixtureData } from './fixtures'
import { assertProjectIDs, fetchedDate, languageMix, projectList, toProject } from './model'

describe('languageMix', () => {
  it('orders by size, then by name, and rounds to whole percent', () => {
    const mix = languageMix([
      { name: 'Shell', bytes: 1000 },
      { name: 'Vue', bytes: 9000 },
      { name: 'Go', bytes: 9000 },
    ])
    expect(mix).toEqual([
      { name: 'Go', share: '47%' },
      { name: 'Vue', share: '47%' },
      { name: 'Shell', share: '5%' },
    ])
  })

  it('shows a share below one percent as <1% and drops a language with no bytes', () => {
    const mix = languageMix([
      { name: 'Go', bytes: 100000 },
      { name: 'Make', bytes: 10 },
      { name: 'Empty', bytes: 0 },
    ])
    expect(mix).toEqual([
      { name: 'Go', share: '100%' },
      { name: 'Make', share: '<1%' },
    ])
  })

  it('returns an empty list for no data, not a placeholder', () => {
    expect(languageMix([])).toEqual([])
  })

  it('does not depend on the input order', () => {
    const a = [
      { name: 'A', bytes: 5 },
      { name: 'B', bytes: 5 },
    ]
    expect(languageMix(a)).toEqual(languageMix([...a].reverse()))
  })
})

describe('projectList', () => {
  it('lists every project of the dataset, sorted by id', () => {
    const ids = projectList(fixtureData, 'en').map((p) => p.id)
    expect(ids).toEqual(['alpha-app', 'no-description', 'no-languages', 'no-release', 'zeta-tool'])
  })

  it('gives the same list for the same projects in any order', () => {
    const shuffled = {
      ...fixtureData,
      projects: { ...fixtureData.projects, en: [...fixtureData.projects.en].reverse() },
    }
    expect(projectList(shuffled, 'en')).toEqual(projectList(fixtureData, 'en'))
  })

  it('takes the descriptions of the asked language', () => {
    const [first] = projectList(fixtureData, 'de')
    expect(first?.description).toBe('An app, the first one. (de)')
  })
})

describe('toProject', () => {
  const byId = (id: string) => {
    const found = projectList(fixtureData, 'en').find((p) => p.id === id)
    if (!found) throw new Error(`no fixture ${id}`)
    return found
  }

  it('carries description, languages, release and the GitHub facts', () => {
    expect(byId('alpha-app')).toMatchObject({
      name: 'alpha-app',
      fullName: 'example/alpha-app',
      description: 'An app, the first one.',
      githubUrl: 'https://github.com/example/alpha-app',
      release: {
        tag: 'v1.2.3',
        url: 'https://github.com/example/alpha-app/releases/tag/v1.2.3',
        date: '2026-10-20',
      },
      topics: ['alpha', 'zeta'],
      license: 'Apache-2.0',
      pushedAt: '2026-11-01',
    })
  })

  it('has no release when there is none', () => {
    expect(byId('no-release').release).toBeNull()
  })

  it('has no description when the cv text is blank, and does not borrow the GitHub one', () => {
    expect(byId('no-description').description).toBeNull()
  })

  it('has no languages and no licence when GitHub knows none', () => {
    const p = byId('no-languages')
    expect(p.languages).toEqual([])
    expect(p.license).toBeNull()
  })

  it('treats a licence GitHub could not identify as none', () => {
    const base = fixtureData.projects.en[0]
    const p = toProject({ ...base, github: { ...base.github, license: 'NOASSERTION' } })
    expect(p.license).toBeNull()
  })
})

describe('fetchedDate', () => {
  it('is the day of the fetch', () => {
    expect(fetchedDate(fixtureData)).toBe('2026-11-18')
  })
})

describe('assertProjectIDs', () => {
  it('accepts hyphenated lower case ids', () => {
    expect(() => assertProjectIDs([{ id: 'a' }, { id: 'b-2' }])).not.toThrow()
  })

  it.each(['', 'Upper', 'a/b', '..', 'a b', '-a', 'a--b'])('rejects the id %j', (id) => {
    expect(() => assertProjectIDs([{ id }])).toThrow(/path segment/)
  })

  it('rejects a repeated id', () => {
    expect(() => assertProjectIDs([{ id: 'a' }, { id: 'a' }])).toThrow(/twice/)
  })
})
