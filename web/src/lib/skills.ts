import type { CvDocument, Skill } from './dataset'

/** projectPath is where a project's page lives. The projects page is built by
 *  its own issue; when it lands, this one function is the only place to
 *  change if its route differs. The fragment keeps the link valid for a
 *  single list page as well as for a page per project. */
export function projectPath(id: string): string {
  return `/projects/#${encodeURIComponent(id)}`
}

/** LEVELS is cv's level scale, highest first: the order the page shows. */
export const LEVELS = ['expert', 'proficient', 'working', 'familiar'] as const

export interface SkillGroup {
  level: string
  skills: Skill[]
}

/** groupByLevel groups skills by level, highest first. A level outside the
 *  scale still gets its own group after the known ones: a skill is never
 *  dropped because cv grew a level. Skills keep the order cv gave them. */
export function groupByLevel(skills: readonly Skill[]): SkillGroup[] {
  const groups = new Map<string, Skill[]>()
  for (const level of LEVELS) {
    groups.set(level, [])
  }
  for (const skill of skills) {
    const list = groups.get(skill.level) ?? []
    list.push(skill)
    groups.set(skill.level, list)
  }
  return [...groups.entries()]
    .filter(([, list]) => list.length > 0)
    .map(([level, list]) => ({ level, skills: list }))
}

/** isExpired says whether a certificate's valid_until lies before now. A month
 *  ("2026-01") is valid through its last day, a date through that day. A value
 *  in no known shape, and null, count as not expired: the page then prints the
 *  value as given instead of asserting something it cannot know. */
export function isExpired(validUntil: string | null, now: Date): boolean {
  if (validUntil === null) {
    return false
  }
  const match = /^(\d{4})-(\d{2})(?:-(\d{2}))?$/.exec(validUntil)
  if (!match) {
    return false
  }
  const year = Number(match[1])
  const month = Number(match[2]) - 1
  const end =
    match[3] === undefined
      ? Date.UTC(year, month + 1, 1)
      : Date.UTC(year, month, Number(match[3]) + 1)
  return now.getTime() >= end
}

export type ResolvedEvidence =
  | { kind: 'project'; id: string; repo: string; href: string }
  | { kind: 'experience'; id: string; employer: string; title: string }
  | {
      kind: 'certificate'
      id: string
      name: string
      issuer: string
      validUntil: string | null
      expired: boolean
    }
  | { kind: 'unresolved'; type: string; ref: string }

/** resolveEvidence follows one skill's evidence list into the entries it
 *  names. A reference that points at nothing, or at a type this page does not
 *  know, stays in the list as "unresolved": the evidence is the point, so it is
 *  shown as given rather than dropped. */
export function resolveEvidence(skill: Skill, doc: CvDocument, now: Date): ResolvedEvidence[] {
  return skill.evidence.map((item): ResolvedEvidence => {
    if (item.type === 'project') {
      const project = doc.projects.find((p) => p.id === item.ref)
      if (project) {
        return {
          kind: 'project',
          id: project.id,
          repo: project.github_repo,
          href: projectPath(project.id),
        }
      }
    } else if (item.type === 'experience') {
      const entry = doc.experience.find((e) => e.id === item.ref)
      if (entry) {
        return { kind: 'experience', id: entry.id, employer: entry.employer, title: entry.title }
      }
    } else if (item.type === 'certificate') {
      const cert = doc.certificates.find((c) => c.id === item.ref)
      if (cert) {
        return {
          kind: 'certificate',
          id: cert.id,
          name: cert.name,
          issuer: cert.issuer,
          validUntil: cert.valid_until,
          expired: isExpired(cert.valid_until, now),
        }
      }
    }
    return { kind: 'unresolved', type: item.type, ref: item.ref }
  })
}
