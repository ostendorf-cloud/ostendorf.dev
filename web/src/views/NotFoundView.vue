<script setup lang="ts">
import { GovuePage } from '@ostendorf-cloud/govue'
</script>

<template>
  <GovuePage
    width="reading"
    title="Page not found"
    lead="The page you were looking for does not exist."
  >
    <p>
      <router-link class="text-accent-text hover:underline" :to="{ name: 'home' }">
        Back to the home page
      </router-link>
    </p>
  </GovuePage>
</template>
