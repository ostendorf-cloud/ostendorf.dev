/** THEME_NAME is this site's one theme: a neutral, dark-first placeholder
 *  palette in src/theme/ostendorf-dev.css, not a real brand yet; see
 *  AGENTS.md. main.ts and the prerender script both start useTheme with
 *  it, so the data-theme attribute the server-rendered markup carries
 *  always matches what the hydrated bundle picks. */
export const THEME_NAME = 'ostendorf-dev'
