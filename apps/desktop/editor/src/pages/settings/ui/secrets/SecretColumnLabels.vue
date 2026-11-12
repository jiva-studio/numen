<script setup lang="ts">
/**
 * The labels over the columns of a card, drawn once above its first row in the grid of the rows.
 * The name label is there only where the rows have the name column.
 */
import { WORDS as words } from '../../words'
import SecretRowFrame from './SecretRowFrame.vue'

defineProps<{ hasName: boolean }>()
</script>

<template>
  <SecretRowFrame :has-name="hasName" class="secret-labels" aria-hidden="true">
    <template #name>
      <span class="secret-labels__text">{{ words.columnName }}</span>
    </template>
    <template #value>
      <span class="secret-labels__text">{{ words.columnValue }}</span>
    </template>
    <template #permissions>
      <span class="secret-labels__text">{{ words.columnUsedBy }}</span>
    </template>
  </SecretRowFrame>
</template>

<style scoped>
.secret-labels :deep(.secret-row) {
  padding-block: 0;
}

.secret-labels__text {
  --pad-inline: 0.125rem;
  --tracking: 0.06em;
  padding-inline: var(--pad-inline);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  letter-spacing: var(--tracking);
  text-transform: uppercase;
}
</style>
