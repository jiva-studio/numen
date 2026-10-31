<script setup lang="ts">
/**
 * The card of a plugin the host suspended: why it stopped, when, and the ways to bring it back.
 */
import { Button } from '@numen/ui'
import type { PluginFailure } from '@numen/plugin-sdk'
import { formatFailedAt } from './state'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  failure: PluginFailure
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
    <span class="plugin-stopped__reason">{{ props.failure.reason }}</span>
    <span class="plugin-stopped__when">{{ formatFailedAt(props.failure.failedAt) }}</span>
    <details v-if="props.failure.detail" class="plugin-stopped__detail">
      <summary>{{ words.pluginDetails }}</summary>
      <code>{{ props.failure.detail }}</code>
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
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas:
    'reason  actions'
    'when    actions'
    'detail  detail';
  column-gap: var(--numen-panel-gap);
  row-gap: 0.125rem;
  align-items: start;
  margin-block-end: var(--settings-apart, 1.75rem);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-alarm);
  border-radius: var(--numen-radius-small);
  color: var(--numen-ink);
}

.plugin-stopped:last-child {
  margin-block-end: 0;
}

.plugin-stopped__reason {
  grid-area: reason;
  overflow-wrap: anywhere;
}

.plugin-stopped__when {
  grid-area: when;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-stopped__detail {
  grid-area: detail;
  margin-block-start: 0.25rem;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-stopped__detail summary {
  cursor: pointer;
}

.plugin-stopped__detail code {
  display: block;
  margin-block-start: 0.25rem;
  font-family: var(--numen-font-mono);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.plugin-stopped__actions {
  grid-area: actions;
  display: flex;
  gap: var(--settings-near, 0.375rem);
}
</style>
