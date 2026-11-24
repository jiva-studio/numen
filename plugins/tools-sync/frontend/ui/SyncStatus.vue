<script setup lang="ts">
/** The state of a vault or a folder in plain words, with a dot that is green, amber or red. */

/* --------------------------------- Props ---------------------------------- */
defineProps<{ text: string; tone: 'ok' | 'warning' | 'alarm' }>()
</script>

<template>
  <span class="sync-status" :class="`sync-status--${tone}`" role="status">
    <span class="sync-status__dot" aria-hidden="true" />
    {{ text }}
  </span>
</template>

<style scoped>
.sync-status {
  display: inline-flex;
  align-items: baseline;
  gap: 0.375rem;
  color: var(--numen-ink);
}

.sync-status__dot {
  flex: none;
  inline-size: 0.5rem;
  block-size: 0.5rem;
  border-radius: var(--numen-radius-pill);
  background-color: var(--numen-positive);
}

.sync-status--warning .sync-status__dot {
  background-color: var(--numen-caution-fg);
}

.sync-status--alarm {
  color: var(--numen-alarm);
}

.sync-status--alarm .sync-status__dot {
  background-color: var(--numen-alarm);
}
</style>
