import { inject, type InjectionKey } from 'vue'

import type { Lang } from '../lib/dataset'

/** The skills page's own message namespace. The site has no language layer
 *  yet (govue's createGovueI18n is not wired in), so this file stays
 *  self contained: when the layer lands, the `skills` object moves into its
 *  catalogues under a `skills.` prefix and localeKey is replaced by the
 *  layer's locale. */
export interface SkillsMessages {
  title: string
  lead: string
  levels: Record<string, string>
  evidence: string
  noEvidence: string
  types: { project: string; experience: string; certificate: string; unresolved: string }
  issuer: string
  validUntil: string
  expired: string
}

export const skills: Record<Lang, SkillsMessages> = {
  en: {
    title: 'Skills and tools',
    lead: 'Each skill shows the level in words and the evidence behind it.',
    levels: {
      expert: 'Expert',
      proficient: 'Proficient',
      working: 'Working knowledge',
      familiar: 'Familiar',
    },
    evidence: 'Evidence',
    noEvidence: 'No evidence listed.',
    types: {
      project: 'Project',
      experience: 'Position',
      certificate: 'Certificate',
      unresolved: 'Reference',
    },
    issuer: 'Issuer',
    validUntil: 'valid until',
    expired: 'Expired',
  },
  de: {
    title: 'Skills und Werkzeuge',
    lead: 'Jeder Skill zeigt die Stufe in Worten und die Belege dahinter.',
    levels: {
      expert: 'Experte',
      proficient: 'Versiert',
      working: 'Im Einsatz',
      familiar: 'Vertraut',
    },
    evidence: 'Belege',
    noEvidence: 'Keine Belege angegeben.',
    types: {
      project: 'Projekt',
      experience: 'Position',
      certificate: 'Zertifikat',
      unresolved: 'Verweis',
    },
    issuer: 'Aussteller',
    validUntil: 'gültig bis',
    expired: 'Abgelaufen',
  },
}

/** localeKey lets the page shell hand the skills page its language. Without
 *  one, the page speaks English, the language of the site's other pages. */
export const localeKey: InjectionKey<Lang> = Symbol('locale')

export function useLocale(): Lang {
  return inject(localeKey, 'en')
}
