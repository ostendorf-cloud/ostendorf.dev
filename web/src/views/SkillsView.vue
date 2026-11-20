<script setup lang="ts">
import { computed } from 'vue'
import { GovueBadge, GovuePage } from '@ostendorf-cloud/govue'

import { skills as messages, useLocale } from '../i18n/skills'
import { useDataset } from '../lib/dataset'
import { groupByLevel, resolveEvidence } from '../lib/skills'

const dataset = useDataset()
const lang = useLocale()
const t = computed(() => messages[lang])

// The build's clock decides "expired", so the prerendered page and its
// hydration cannot disagree about a certificate that lapsed in between.
const now = new Date(dataset.generated_at)

const doc = computed(() => {
  const found = dataset.cv[lang]
  if (!found) {
    throw new Error(`skills: the dataset has no document for language "${lang}"`)
  }
  return found
})

const groups = computed(() =>
  groupByLevel(doc.value.skills).map((group) => ({
    ...group,
    label: t.value.levels[group.level] ?? group.level,
    skills: group.skills.map((skill) => ({
      ...skill,
      evidence: resolveEvidence(skill, doc.value, now),
    })),
  })),
)
</script>

<template>
  <GovuePage :title="t.title" :lead="t.lead">
    <section v-for="group in groups" :key="group.level" class="flex flex-col gap-3">
      <h2 class="text-xl font-semibold text-text">{{ group.label }}</h2>
      <ul class="flex flex-col gap-4">
        <li v-for="skill in group.skills" :key="skill.id" class="flex flex-col gap-1">
          <h3 class="font-medium text-text">{{ skill.name }}</h3>
          <p v-if="skill.evidence.length === 0" class="text-sm text-text-muted">
            {{ t.noEvidence }}
          </p>
          <template v-else>
            <p class="text-sm text-text-muted">{{ t.evidence }}</p>
            <ul class="flex flex-col gap-1 text-sm">
              <li v-for="(item, index) in skill.evidence" :key="index" class="text-text">
                <template v-if="item.kind === 'project'">
                  <span class="text-text-muted">{{ t.types.project }}:</span>
                  <a
                    :href="item.href"
                    class="font-medium text-accent-text underline-offset-2 hover:underline"
                    >{{ item.repo }}</a
                  >
                </template>
                <template v-else-if="item.kind === 'experience'">
                  <span class="text-text-muted">{{ t.types.experience }}:</span>
                  {{ item.title }}, {{ item.employer }}
                </template>
                <template v-else-if="item.kind === 'certificate'">
                  <span class="text-text-muted">{{ t.types.certificate }}:</span>
                  {{ item.name }}
                  <span class="text-text-muted">
                    ({{ t.issuer }}: {{ item.issuer
                    }}<template v-if="item.validUntil !== null"
                      >, {{ t.validUntil }} {{ item.validUntil }}</template
                    >)
                  </span>
                  <GovueBadge v-if="item.expired" kind="danger">{{ t.expired }}</GovueBadge>
                </template>
                <template v-else>
                  <span class="text-text-muted">{{ t.types.unresolved }}:</span>
                  {{ item.type }} {{ item.ref }}
                </template>
              </li>
            </ul>
          </template>
        </li>
      </ul>
    </section>
  </GovuePage>
</template>
