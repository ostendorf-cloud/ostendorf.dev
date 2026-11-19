import { createApp } from 'vue'
import { createWebHistory } from 'vue-router'
import { useTheme } from '@ostendorf-cloud/govue'

import App from './App.vue'
import { createSiteI18n, followRoute, localeOfPath } from './lib/language'
import { THEME_NAME } from './lib/theme'
import { createAppRouter } from './router'
import './styles.css'

// The theme never varies: this site ships exactly one brand, still a
// neutral placeholder (see src/theme/ostendorf-dev.css). Mode starts at
// "system", the visitor's own choice only overrides it once the header's
// theme toggle changes it, matching useTheme's own documented default.
useTheme({ theme: THEME_NAME })

const app = createApp(App)
const i18n = createSiteI18n(localeOfPath(window.location.pathname))
const router = createAppRouter(createWebHistory())
followRoute(router, i18n)
app.use(i18n)
app.use(router)
app.mount('#app')
