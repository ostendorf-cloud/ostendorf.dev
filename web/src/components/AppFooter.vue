<script setup lang="ts">
import { computed } from 'vue'
import { GovueFooter } from '@ostendorf-cloud/govue'

// No link to simon-ostendorf.de or ostendorf.tech here, on purpose: see
// AGENTS.md for the decision and its reasoning. simon-ostendorf.de is
// the hub that already carries the outbound links between all three
// sites, so this footer does not restate them.
const note = computed(() => `© ${new Date().getFullYear()} Simon Ostendorf`)
</script>

<template>
  <GovueFooter :note="note" />
</template>
