<script setup lang="ts">
/** Labelled values of a link as rows of a name and its value: a device name, a fingerprint. */

import { Fingerprint } from '@numen/ui'
import type { Fact } from '../../lib/types'

defineProps<{ facts: readonly Fact[] }>()
</script>

<template>
  <dl class="link-facts">
    <div v-for="fact in facts" :key="fact.label" class="link-facts__row">
      <dt class="link-facts__label">{{ fact.label }}</dt>
      <dd class="link-facts__value" :class="{ 'link-facts__value--mono': fact.isMono }">
        <Fingerprint v-if="fact.isMono" :value="fact.value" />
        <template v-else>{{ fact.value }}</template>
      </dd>
    </div>
  </dl>
</template>

<style scoped>
.link-facts {
  display: flex;
  flex-direction: column;
  gap: var(--numen-card-inset);
  margin: 0;
}

.link-facts__row {
  display: grid;
  grid-template-columns: 8rem minmax(0, 1fr);
  gap: var(--numen-inset-wide);
}

.link-facts__label {
  color: var(--numen-hushed);
}

.link-facts__value {
  margin: 0;
  overflow-wrap: anywhere;
}

.link-facts__value--mono {
  font-family: var(--numen-font-mono);
}
</style>
