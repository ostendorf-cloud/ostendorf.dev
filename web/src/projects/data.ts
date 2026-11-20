import { inject, type InjectionKey } from 'vue'

// The virtual module web/projectsData.ts writes.
import embedded from 'virtual:projects-data'

import { projectList, type Project } from './model'
import type { ProjectLocale, ProjectsData } from './types'

/** projectsKey lets a test, or a later caller, provide its own data instead
 *  of the generated dataset. */
export const projectsKey: InjectionKey<ProjectsData> = Symbol('projects-data')

/** projectLocaleKey names the language of the descriptions. The site is
 *  English only until its language layer lands, so the default is English. */
export const projectLocaleKey: InjectionKey<ProjectLocale> = Symbol('project-locale')

/** embeddedProjects is the generated dataset's projects, compiled into the
 *  bundle. It fails when the dataset was not built, which cannot be a state
 *  worth rendering: a page without its projects would only look empty. */
export function embeddedProjects(): ProjectsData {
  if (embedded === null) {
    throw new Error(
      'projects: internal/content/generated/dataset.json is missing, run task content:build',
    )
  }
  return embedded
}

export interface ProjectsState {
  data: ProjectsData
  locale: ProjectLocale
  projects: Project[]
}

/** useProjects is what both project views read. Nothing here makes a request:
 *  the data is part of the bundle. */
export function useProjects(): ProjectsState {
  const data = inject(projectsKey, null) ?? embeddedProjects()
  const locale = inject(projectLocaleKey, 'en')
  return { data, locale, projects: projectList(data, locale) }
}
