import { createGovueI18n, localeFromPath, type LocaleCode } from '@ostendorf-cloud/govue'

/** localeOfPath is the language of a served path: the legal pages live under
 *  "/de/" and "/en/", every other page, the home page included, is English. */
export function localeOfPath(path: string): LocaleCode {
  return localeFromPath(path) ?? 'en'
}

/** createSiteI18n is govue's language layer for one page. The language comes
 *  from the path alone and nothing is stored: the visitor's choice is the
 *  address, so no cookie and no local storage entry is written for it. */
export function createSiteI18n(locale: LocaleCode) {
  return createGovueI18n({ locale, storage: { kind: 'none' } })
}
