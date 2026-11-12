<script setup lang="ts">
/**
 * The raised card that holds a page of settings rows. The settings of a plugin and the system
 * settings share it. The rows inside are separated by hairlines, and the last one has none.
 */
</script>

<template>
  <div class="plugin-settings-card">
    <slot />
  </div>
</template>

<style scoped>
.plugin-settings-card {
  padding-block: calc(var(--settings-card-pad, 1rem) - var(--settings-row-pad, 0.75rem));
  --card-spacing: 1.25rem;
  padding-inline: var(--settings-card-pad, 1rem);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-settings-card + .plugin-settings-card {
  margin-block-start: var(--card-spacing);
}

.plugin-settings-card :deep(.settings__row:last-child) {
  border-block-end: none;
}
</style>
