import { createApp } from 'vue'
import { useTheme } from '@ostendorf-cloud/govue'

import App from './App.vue'
import './styles.css'

// The theme never varies: this site ships exactly one brand, still a
// neutral placeholder (see src/theme/ostendorf-dev.css). Mode starts at
// "system", the visitor's own choice only overrides it once a toggle
// exists, matching useTheme's own documented default.
useTheme({ theme: 'ostendorf-dev' })

createApp(App).mount('#app')
