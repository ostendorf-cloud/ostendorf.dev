import type { ProjectLocale } from './types'

/** The texts of the project pages, one table per language. A key is a plain
 *  property so a missing translation is a type error. The site shows English
 *  until its language layer lands; the German table is ready for it. */
export interface ProjectMessages {
  indexTitle: string
  indexLead: string
  dataNote: (date: string) => string
  staleNote: string
  details: string
  backToIndex: string
  onGitHub: string
  latestRelease: string
  languages: string
  stars: string
  forks: string
  license: string
  lastPush: string
  topics: string
  archived: string
  emptyTitle: string
  emptyBody: string
  notFoundTitle: string
  notFoundBody: string
}

export const messages: Record<ProjectLocale, ProjectMessages> = {
  en: {
    indexTitle: 'Projects',
    indexLead: 'What I build, with what GitHub knows about each project.',
    dataNote: (date) => `GitHub data as of ${date}.`,
    staleNote: 'This GitHub data is older than usual: the last refresh failed.',
    details: 'Details',
    backToIndex: 'All projects',
    onGitHub: 'View on GitHub',
    latestRelease: 'Latest release',
    languages: 'Languages',
    stars: 'Stars',
    forks: 'Forks',
    license: 'License',
    lastPush: 'Last push',
    topics: 'Topics',
    archived: 'Archived',
    emptyTitle: 'No projects yet',
    emptyBody: 'The list is filled from the cv repository and GitHub at build time.',
    notFoundTitle: 'Project not found',
    notFoundBody: 'No project has this address.',
  },
  de: {
    indexTitle: 'Projekte',
    indexLead: 'Woran ich baue, mit den Angaben, die GitHub zu jedem Projekt kennt.',
    dataNote: (date) => `GitHub-Daten vom ${date}.`,
    staleNote:
      'Diese GitHub-Daten sind älter als üblich: die letzte Aktualisierung ist fehlgeschlagen.',
    details: 'Details',
    backToIndex: 'Alle Projekte',
    onGitHub: 'Auf GitHub ansehen',
    latestRelease: 'Letztes Release',
    languages: 'Sprachen',
    stars: 'Sterne',
    forks: 'Forks',
    license: 'Lizenz',
    lastPush: 'Letzter Push',
    topics: 'Themen',
    archived: 'Archiviert',
    emptyTitle: 'Noch keine Projekte',
    emptyBody: 'Die Liste wird beim Build aus dem cv-Repository und von GitHub gefüllt.',
    notFoundTitle: 'Projekt nicht gefunden',
    notFoundBody: 'Unter dieser Adresse gibt es kein Projekt.',
  },
}
