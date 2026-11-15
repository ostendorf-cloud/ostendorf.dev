<script setup lang="ts">
import { GovueSkipLink } from '@ostendorf-cloud/govue'

import AppFooter from './components/AppFooter.vue'
import AppHeader from './components/AppHeader.vue'
</script>

<template>
  <GovueSkipLink>Skip to content</GovueSkipLink>
  <AppHeader />
  <router-view />
  <AppFooter />
</template>
