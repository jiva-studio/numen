<script setup lang="ts">
/**
 * The card of a plugin the host suspended: why it stopped, and the ways to bring it back.
 */
import { Button } from '@numen/ui'
import type { PluginHealth } from './pluginInfo'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  item: PluginHealth
}>()

const emit = defineEmits<{
  (e: 'retry'): void
  (e: 'disable'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onRetry(): void {
  emit('retry')
}

function onDisable(): void {
  emit('disable')
}
</script>

<template>
  <div class="plugin-stopped" role="group" :aria-label="words.pluginState.stopped">
    <span class="plugin-stopped__reason">{{ props.item.text }}</span>
    <details v-if="props.item.detail" class="plugin-stopped__detail">
      <summary>{{ words.pluginDetails }}</summary>
      <code>{{ props.item.detail }}</code>
    </details>
    <div class="plugin-stopped__actions">
      <Button size="small" @click="onRetry">{{ words.pluginRetry }}</Button>
      <Button variant="outline" size="small" @click="onDisable">
        {{ words.pluginDisable }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.plugin-stopped {
  --stopped-row-gap: 0.125rem;
  --stopped-spacing: 1.75rem;
  --stopped-pad-block: 0.5rem;
  --stopped-pad-inline: 0.75rem;
  --stopped-detail-offset: 0.25rem;
  --stopped-action-gap: 0.375rem;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas:
    'reason  actions'
    'detail  detail';
  column-gap: var(--numen-panel-gap);
  row-gap: var(--stopped-row-gap);
  align-items: start;
  margin-block-end: var(--stopped-spacing);
  padding: var(--stopped-pad-block) var(--stopped-pad-inline);
  border: var(--numen-stroke) solid var(--numen-alarm);
  border-radius: var(--numen-radius-tight);
  color: var(--numen-ink);
}

.plugin-stopped:last-child {
  margin-block-end: 0;
}

.plugin-stopped__reason {
  grid-area: reason;
  overflow-wrap: anywhere;
}

.plugin-stopped__detail {
  grid-area: detail;
  margin-block-start: var(--stopped-detail-offset);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-stopped__detail summary {
  cursor: pointer;
}

.plugin-stopped__detail code {
  display: block;
  margin-block-start: var(--stopped-detail-offset);
  font-family: var(--numen-font-mono);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.plugin-stopped__actions {
  grid-area: actions;
  display: flex;
  gap: var(--stopped-action-gap);
}
</style>
