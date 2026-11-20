<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { GovueBadge, GovuePage } from '@ostendorf-cloud/govue'

import { useProjects } from '../projects/data'
import { messages } from '../projects/messages'
import ProjectFacts from '../projects/ProjectFacts.vue'

const route = useRoute()
const { locale, projects } = useProjects()
const text = messages[locale]
const project = computed(() => projects.find((p) => p.id === route.params.id))
</script>

<template>
  <GovuePage v-if="project" :title="project.name">
    <p v-if="project.archived">
      <GovueBadge kind="warning">{{ text.archived }}</GovueBadge>
    </p>
    <p v-if="project.description" class="text-lg text-text">{{ project.description }}</p>
    <ProjectFacts :project="project" :text="text" detailed />
    <p>
      <a
        :href="project.githubUrl"
        rel="noopener noreferrer"
        class="font-medium text-accent-text underline-offset-2 hover:underline"
        >{{ text.onGitHub }}</a
      >
    </p>
    <p>
      <router-link
        :to="{ name: 'projects' }"
        class="font-medium text-accent-text underline-offset-2 hover:underline"
        >{{ text.backToIndex }}</router-link
      >
    </p>
  </GovuePage>
  <GovuePage v-else :title="text.notFoundTitle" :lead="text.notFoundBody">
    <p>
      <router-link
        :to="{ name: 'projects' }"
        class="font-medium text-accent-text underline-offset-2 hover:underline"
        >{{ text.backToIndex }}</router-link
      >
    </p>
  </GovuePage>
</template>
