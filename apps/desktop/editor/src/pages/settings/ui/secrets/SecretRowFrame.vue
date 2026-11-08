<script setup lang="ts">
/**
 * The columns every secret row has, saved or not yet created: the name, where the group holds more
 * than one secret, the value, the plugins that may read it, and the actions.
 */
defineProps<{
  /** Whether the row has the name column. */
  hasName: boolean
}>()
</script>

<template>
  <li class="secret-row" :class="{ 'secret-row--named': hasName }">
    <slot v-if="hasName" name="name" />
    <slot name="value" />
    <slot name="permissions" />
    <div class="secret-row__actions">
      <slot name="actions" />
    </div>
  </li>
</template>

<style scoped>
.secret-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 12rem var(--numen-action-size);
  align-items: center;
  gap: 0.5rem;
  padding-block: 0.375rem;
  list-style: none;
}

.secret-row--named {
  grid-template-columns: 11rem minmax(0, 1fr) 12rem var(--numen-action-size);
}

.secret-row__actions {
  display: flex;
  gap: 0.125rem;
  min-inline-size: 1.75rem;
}

:slotted(.secret-action) {
  border-radius: var(--numen-radius);
}
</style>
