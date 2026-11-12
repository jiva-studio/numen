<script setup lang="ts">
/**
 * The columns every secret row has, saved or not yet created: the name, where the group holds more
 * than one secret, the value, the plugins that may read it, and the actions. A row with a label is
 * a group of controls of that name.
 */
defineProps<{
  hasName: boolean
  label?: string
}>()
</script>

<template>
  <li class="secret-item">
    <div
      :role="label === undefined ? undefined : 'group'"
      :aria-label="label"
      class="secret-row"
      :class="{ 'secret-row--named': hasName }"
    >
      <slot v-if="hasName" name="name" />
      <slot name="value" />
      <slot name="permissions" />
      <div class="secret-row__actions">
        <slot name="actions" />
      </div>
    </div>
  </li>
</template>

<style scoped>
.secret-item {
  list-style: none;
}

.secret-row {
  --secret-name-width: 11rem;
  --secret-readers-width: 12rem;

  --gap: 0.5rem;
  --pad-block: 0.375rem;
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--secret-readers-width) var(--numen-action-size);
  align-items: start;
  gap: var(--gap);
  padding-block: var(--pad-block);
}

.secret-row--named {
  grid-template-columns:
    var(--secret-name-width) minmax(0, 1fr) var(--secret-readers-width)
    var(--numen-action-size);
}

.secret-row__actions {
  --gap: 0.125rem;
  --min-width: 1.75rem;
  display: flex;
  gap: var(--gap);
  min-inline-size: var(--min-width);
}
</style>
