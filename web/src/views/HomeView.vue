<script setup lang="ts">
import { GovuePage } from '@ostendorf-cloud/govue'

interface ProfileLink {
  label: string
  handle: string
  href: string
}

// The actual contact and profile links old_ostendorf.dev carried, kept
// verbatim; see AGENTS.md. No other request leaves the browser on load.
const links: ProfileLink[] = [
  {
    label: 'GitHub',
    handle: 'github.com/simonostendorf',
    href: 'https://github.com/simonostendorf',
  },
  {
    label: 'LinkedIn',
    handle: 'in/simonostendorf',
    href: 'https://www.linkedin.com/in/simonostendorf',
  },
  {
    label: 'Email',
    handle: 'simon@ostendorf.dev',
    href: 'mailto:simon@ostendorf.dev',
  },
]
</script>

<template>
  <GovuePage title="Simon Ostendorf">
    <p class="text-text">
      DevOps &amp; Cloud Engineer at
      <a
        href="https://stackmeister.com"
        rel="noopener noreferrer"
        class="font-medium text-accent-text underline-offset-2 hover:underline"
        >Stackmeister</a
      >.
    </p>
    <p class="text-text-muted">
      This is the developer site of Simon Ostendorf: projects, skills and a contact form are on
      their way. The links below reach me in the meantime.
    </p>
    <ul class="flex flex-col gap-2">
      <li v-for="link in links" :key="link.href">
        <a
          :href="link.href"
          rel="noopener noreferrer"
          class="font-medium text-accent-text underline-offset-2 hover:underline"
          >{{ link.label }}</a
        >
        <span class="text-text-muted">— {{ link.handle }}</span>
      </li>
    </ul>
  </GovuePage>
</template>
