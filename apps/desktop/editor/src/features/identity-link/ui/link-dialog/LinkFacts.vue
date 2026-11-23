<script setup lang="ts">
/** Labelled values of a link: a name, a fingerprint, a code. */

import type { Fact } from '../../lib/types'

defineProps<{ facts: readonly Fact[] }>()
</script>

<template>
  <dl class="link-facts">
    <template v-for="fact in facts" :key="fact.label">
      <dt>{{ fact.label }}</dt>
      <dd :class="{ 'link-facts__mono': fact.isMono }">{{ fact.value }}</dd>
    </template>
  </dl>
</template>

<style scoped>
.link-facts {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.375rem 1rem;
  margin: 0;
}

.link-facts dd {
  margin: 0;
}

.link-facts__mono {
  font-family: var(--numen-font-mono);
}
</style>
