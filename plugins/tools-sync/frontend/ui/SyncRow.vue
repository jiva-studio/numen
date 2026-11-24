<script setup lang="ts">
/**
 * A row of a card of the sync page: the name with the lines under it on the left, and what the person
 * does with it on the right. Rows follow each other with a rule between them.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{ name: string }>()
</script>

<template>
  <li class="sync-row">
    <div class="sync-row__label">
      <span class="sync-row__name">{{ name }}</span>
      <slot />
    </div>
    <div v-if="$slots['actions']" class="sync-row__actions"><slot name="actions" /></div>
  </li>
</template>

<style scoped>
.sync-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.5rem 1rem;
  box-sizing: border-box;
  min-block-size: 3.25rem;
  padding-block: 0.75rem;
}

.sync-row__label {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
}

.sync-row__name {
  font-weight: 500;
  overflow-wrap: anywhere;
}

.sync-row__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
