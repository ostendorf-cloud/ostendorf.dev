<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { GovueHeader, GovueNav, type NavItem } from '@ostendorf-cloud/govue'

import ThemeToggle from './ThemeToggle.vue'

const route = useRoute()

// Skills and the uses page add their own nav item once their own issue lands.
const items = computed<NavItem[]>(() => [
  { href: '/', label: 'Home', current: route.name === 'home' },
  {
    href: '/projects/',
    label: 'Projects',
    current: route.name === 'projects' || route.name === 'project',
  },
])
</script>

<template>
  <GovueHeader>
    <template #brand>
      <router-link :to="{ name: 'home' }" class="inline-flex min-h-11 items-center">
        Simon Ostendorf
      </router-link>
    </template>
    <GovueNav :items="items" aria-label="Main" />
    <template #actions>
      <ThemeToggle />
    </template>
  </GovueHeader>
</template>
