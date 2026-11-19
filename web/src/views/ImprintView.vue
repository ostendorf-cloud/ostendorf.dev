<script setup lang="ts">
import { GovueLegalPage, useLanguage } from '@ostendorf-cloud/govue'

import { operator } from '../legal/operator'

// The site's own imprint text. govue supplies the page shell and no words.
// The facts come from legal/operator.ts, which the privacy text shares.
// There is no dispute resolution section, on purpose.
// No navigate prop: the language switch is a full page load, every language
// is its own prerendered page with its own head.
const { locale } = useLanguage()
</script>

<template>
  <GovueLegalPage page="imprint" :skip-link="false" back-href="/">
    <div v-if="locale === 'de'" class="flex flex-col gap-4 text-text">
      <p>
        {{ operator.name }}<br />
        {{ operator.street }}<br />
        {{ operator.city }}
      </p>

      <h2 class="text-xl font-semibold">Kontakt</h2>
      <p>
        Telefon:
        <a class="text-accent-text hover:underline" :href="operator.phoneHref">{{
          operator.phone
        }}</a><br />
        E-Mail:
        <a class="text-accent-text hover:underline" :href="`mailto:${operator.email}`">{{
          operator.email
        }}</a>
      </p>

      <h2 class="text-xl font-semibold">Umsatzsteuer-ID</h2>
      <p>Umsatzsteuer-Identifikationsnummer gemäß §27a Umsatzsteuergesetz: {{ operator.vatId }}</p>
    </div>

    <div v-else class="flex flex-col gap-4 text-text">
      <p class="text-text-muted">
        The German version of this page is the legally binding one.
        <a class="text-accent-text hover:underline" href="/de/imprint">View the German version</a>
      </p>

      <p>
        {{ operator.name }}<br />
        {{ operator.street }}<br />
        {{ operator.city }}, Germany
      </p>

      <h2 class="text-xl font-semibold">Contact</h2>
      <p>
        Phone:
        <a class="text-accent-text hover:underline" :href="operator.phoneHref">{{
          operator.phone
        }}</a><br />
        Email:
        <a class="text-accent-text hover:underline" :href="`mailto:${operator.email}`">{{
          operator.email
        }}</a>
      </p>

      <h2 class="text-xl font-semibold">VAT ID</h2>
      <p>VAT identification number per Section 27a of the German VAT Act: {{ operator.vatId }}</p>
    </div>
  </GovueLegalPage>
</template>
