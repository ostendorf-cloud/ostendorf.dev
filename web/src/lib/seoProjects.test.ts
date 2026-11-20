import { describe, expect, it } from 'vitest'

import { canonical, fullTitle, PROJECT, projectCanonical, projectMeta, projectPath } from './seo'

describe('the projects index', () => {
  it('has its own title, description and canonical URL', () => {
    expect(canonical('projects')).toMatch(/\/projects\/$/)
    expect(fullTitle('projects', { title: 'Projects', description: '' })).toBe(
      'Projects – Simon Ostendorf',
    )
  })
})

describe('projectMeta', () => {
  it('uses the project name as title and its description as description', () => {
    expect(projectMeta({ name: 'demo', description: 'Does one thing.' })).toEqual({
      title: 'demo',
      description: 'Does one thing.',
    })
  })

  it('writes a plain sentence for a project with no description', () => {
    const meta = projectMeta({ name: 'demo', description: null })
    expect(meta.description).toBe('demo, a project of Simon Ostendorf.')
  })

  it('cuts a long description at a word boundary', () => {
    const long = 'word '.repeat(60).trim()
    const { description } = projectMeta({ name: 'demo', description: long })
    expect(description.length).toBeLessThanOrEqual(158)
    expect(description).toMatch(/word\.\.\.$/)
  })

  it('gives two projects different titles and descriptions', () => {
    const a = projectMeta({ name: 'a', description: 'First.' })
    const b = projectMeta({ name: 'b', description: 'Second.' })
    expect(a).not.toEqual(b)
  })
})

describe('project paths', () => {
  it('builds the path and canonical URL of a project page', () => {
    expect(projectPath('demo')).toBe('/projects/demo/')
    expect(projectCanonical('demo')).toMatch(/^https?:\/\/.+\/projects\/demo\/$/)
  })

  it('titles a project page like every other page but home', () => {
    expect(fullTitle(PROJECT, projectMeta({ name: 'demo', description: null }))).toBe(
      'demo – Simon Ostendorf',
    )
  })
})
