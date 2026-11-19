import {
  createGovueI18n,
  localeFromPath,
  type GovueI18n,
  type LocaleCode,
} from '@ostendorf-cloud/govue'
import type { Router } from 'vue-router'

/** localeOfPath is the language of a served path: the legal pages live under
 *  "/de/" and "/en/", every other page, the home page included, is English. */
export function localeOfPath(path: string): LocaleCode {
  return localeFromPath(path) ?? 'en'
}

/** createSiteI18n is govue's language layer for one page. The language comes
 *  from the path alone and nothing is stored: the visitor's choice is the
 *  address, so no cookie and no local storage entry is written for it. */
export function createSiteI18n(locale: LocaleCode): GovueI18n {
  return createGovueI18n({ locale, storage: { kind: 'none' } })
}

/** followRoute keeps the language equal to the address after every client
 *  side navigation: the browser's Back button, and a link such as the
 *  header's brand link from a German page to the English home page. The
 *  language switch itself needs no help, it is a full page load. */
export function followRoute(router: Router, i18n: GovueI18n): void {
  router.afterEach((to) => {
    i18n.language.setLocale(localeOfPath(to.path), { persist: false })
  })
}
