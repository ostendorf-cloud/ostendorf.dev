<script setup lang="ts">
import { GovueBadge } from '@ostendorf-cloud/govue'

import type { Project } from './model'
import type { ProjectMessages } from './messages'

// Every row is v-if'd on its own data: a project without a release, without
// language data or without a licence has no row for it, no gap and no dash.
defineProps<{
  project: Project
  text: ProjectMessages
  /** The project page adds the rows the index leaves out. */
  detailed?: boolean
}>()
</script>

<template>
  <dl class="flex flex-col gap-2 text-sm">
    <div v-if="project.release" class="flex flex-wrap gap-x-2">
      <dt class="text-text-muted">{{ text.latestRelease }}</dt>
      <dd>
        <a
          :href="project.release.url"
          rel="noopener noreferrer"
          class="font-medium text-accent-text underline-offset-2 hover:underline"
          >{{ project.release.tag }}</a>{{ ' '
        }}<time :datetime="project.release.date" class="text-text-muted">{{
          project.release.date
        }}</time>
      </dd>
    </div>
    <div v-if="project.languages.length > 0" class="flex flex-wrap gap-x-2">
      <dt class="text-text-muted">{{ text.languages }}</dt>
      <dd>
        <ul class="flex flex-wrap gap-1">
          <li v-for="language in project.languages" :key="language.name">
            <GovueBadge>{{ language.name }} {{ language.share }}</GovueBadge>
          </li>
        </ul>
      </dd>
    </div>
    <template v-if="detailed">
      <div class="flex flex-wrap gap-x-2">
        <dt class="text-text-muted">{{ text.stars }}</dt>
        <dd>{{ project.stars }}</dd>
      </div>
      <div class="flex flex-wrap gap-x-2">
        <dt class="text-text-muted">{{ text.forks }}</dt>
        <dd>{{ project.forks }}</dd>
      </div>
      <div v-if="project.license" class="flex flex-wrap gap-x-2">
        <dt class="text-text-muted">{{ text.license }}</dt>
        <dd>{{ project.license }}</dd>
      </div>
      <div v-if="project.pushedAt" class="flex flex-wrap gap-x-2">
        <dt class="text-text-muted">{{ text.lastPush }}</dt>
        <dd>
          <time :datetime="project.pushedAt">{{ project.pushedAt }}</time>
        </dd>
      </div>
      <div v-if="project.topics.length > 0" class="flex flex-wrap gap-x-2">
        <dt class="text-text-muted">{{ text.topics }}</dt>
        <dd>
          <ul class="flex flex-wrap gap-1">
            <li v-for="topic in project.topics" :key="topic">
              <GovueBadge kind="info">{{ topic }}</GovueBadge>
            </li>
          </ul>
        </dd>
      </div>
    </template>
  </dl>
</template>
