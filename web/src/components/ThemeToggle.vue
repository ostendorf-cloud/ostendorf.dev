<script setup lang="ts">
import { computed } from 'vue'
import { GovueButton, useTheme } from '@ostendorf-cloud/govue'

/**
 * The header's light/dark/system toggle. useTheme's mode already has
 * three states, light, dark and system, in that cycling order; this
 * button only ever cycles through them. The label names the mode a
 * click switches to, the next one in that order, not the one currently
 * active: an accessible name on a button describes what activating it
 * does.
 */
const cycleOrder = ['light', 'dark', 'system'] as const

const labels: Record<(typeof cycleOrder)[number], string> = {
  light: 'Switch to light mode',
  dark: 'Switch to dark mode',
  system: 'Follow system theme',
}

const { mode, resolvedMode, cycleMode } = useTheme()

const nextMode = computed(
  () => cycleOrder[(cycleOrder.indexOf(mode.value) + 1) % cycleOrder.length],
)
const label = computed(() => labels[nextMode.value])
</script>

<template>
  <GovueButton variant="quiet" :title="label" :aria-label="label" @click="cycleMode">
    <svg
      v-if="resolvedMode === 'dark'"
      class="size-4 shrink-0"
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z"
        fill="none"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <svg v-else class="size-4 shrink-0" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <circle cx="12" cy="12" r="4.5" fill="none" stroke="currentColor" stroke-width="1.7" />
      <path
        d="M12 2.5v2.5M12 19v2.5M4.2 4.2l1.8 1.8M18 18l1.8 1.8M2.5 12H5M19 12h2.5M4.2 19.8 6 18M18 6l1.8-1.8"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
      />
    </svg>
  </GovueButton>
</template>
