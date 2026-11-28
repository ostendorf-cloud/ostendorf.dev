<script setup lang="ts">
import { GovuePage } from '@ostendorf-cloud/govue'
</script>

<template>
  <GovuePage
    width="reading"
    title="Temporarily unavailable"
    lead="This page reads its content from a service that does not answer right now."
  >
    <p>Please try again in a minute.</p>
    <p>
      <router-link class="text-accent-text hover:underline" :to="{ name: 'home' }">
        Back to the home page
      </router-link>
    </p>
  </GovuePage>
</template>
