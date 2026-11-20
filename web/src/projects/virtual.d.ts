// The module web/projectsData.ts provides: the projects of the generated
// dataset, or null when "task content:build" has not run.
declare module 'virtual:projects-data' {
  import type { ProjectsData } from '@/projects/types'

  const data: ProjectsData | null
  export default data
}
