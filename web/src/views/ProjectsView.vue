<script setup lang="ts">
import { GovueBadge, GovueCard, GovueEmptyState, GovuePage } from '@ostendorf-cloud/govue'

import { useProjects } from '../projects/data'
import { messages } from '../projects/messages'
import { fetchedDate } from '../projects/model'
import ProjectFacts from '../projects/ProjectFacts.vue'

const { data, locale, projects } = useProjects()
const text = messages[locale]
const fetched = fetchedDate(data)
</script>

<template>
  <GovuePage width="shell" :title="text.indexTitle" :lead="text.indexLead">
    <GovueEmptyState
      v-if="projects.length === 0"
      :title="text.emptyTitle"
      :description="text.emptyBody"
    />
    <ul v-else class="grid list-none gap-4 p-0 md:grid-cols-2">
      <li v-for="project in projects" :key="project.id" class="flex">
        <GovueCard as="article" class="w-full">
          <template #header>
            <h2 class="flex flex-wrap items-center gap-2 text-lg font-semibold">
              <router-link
                :to="{ name: 'project', params: { id: project.id } }"
                class="text-accent-text underline-offset-2 hover:underline"
                >{{ project.name }}</router-link
              >
              <GovueBadge v-if="project.archived" kind="warning">{{ text.archived }}</GovueBadge>
            </h2>
          </template>
          <div class="flex flex-col gap-3">
            <p v-if="project.description" class="text-text">{{ project.description }}</p>
            <ProjectFacts :project="project" :text="text" />
          </div>
        </GovueCard>
      </li>
    </ul>
    <p v-if="fetched" class="text-sm text-text-muted">{{ text.dataNote(fetched) }}</p>
    <p v-if="data.stale" class="text-sm text-warning-text">{{ text.staleNote }}</p>
  </GovuePage>
</template>
