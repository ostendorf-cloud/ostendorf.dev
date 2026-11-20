import { describe, expect, it } from 'vitest'

import { fixtureDataset } from './skills.fixture'
import { groupByLevel, isExpired, projectPath, resolveEvidence } from './skills'

const now = new Date('2026-06-15T00:00:00Z')

describe('isExpired', () => {
  it.each([
    ['2026-05', true],
    ['2026-06', false],
    ['2026-07', false],
    ['2026-06-14', true],
    ['2026-06-15', false],
    [null, false],
    ['soon', false],
  ])('judges %s as expired=%s', (validUntil, want) => {
    expect(isExpired(validUntil, now)).toBe(want)
  })
})

describe('groupByLevel', () => {
  const skills = fixtureDataset.cv.en.skills

  it('orders the groups from expert down to familiar', () => {
    expect(groupByLevel(skills).map((g) => g.level)).toEqual([
      'expert',
      'proficient',
      'working',
      'familiar',
    ])
  })

  it('keeps cv order inside a group', () => {
    const proficient = groupByLevel(skills).find((g) => g.level === 'proficient')
    expect(proficient?.skills.map((s) => s.id)).toEqual(['s-tf', 's-gone'])
  })

  it('keeps a skill whose level is not on the scale', () => {
    const odd = { id: 'x', name: 'X', level: 'guru', evidence: [] }
    const groups = groupByLevel([...skills, odd])
    expect(groups[groups.length - 1]).toEqual({ level: 'guru', skills: [odd] })
  })
})

describe('resolveEvidence', () => {
  const doc = fixtureDataset.cv.en

  it('links a project through projectPath', () => {
    const [item] = resolveEvidence(doc.skills[3], doc, now)
    expect(item.kind).toBe('certificate')
    const project = resolveEvidence(doc.skills[0], doc, now)[2]
    expect(project).toMatchObject({ kind: 'project', href: projectPath('proj-1') })
  })

  it('marks only the lapsed certificate expired', () => {
    const items = resolveEvidence(doc.skills[0], doc, now)
    expect(items[0]).toMatchObject({ kind: 'certificate', issuer: 'Old Body en', expired: true })
    expect(items[1]).toMatchObject({ kind: 'certificate', expired: false })
  })

  it('keeps a reference that points at nothing', () => {
    expect(resolveEvidence(doc.skills[4], doc, now)).toEqual([
      { kind: 'unresolved', type: 'project', ref: 'proj-missing' },
    ])
  })
})
