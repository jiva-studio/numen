<script setup lang="ts">
/**
 * The bar of the worst health item of a plugin on its page: what does not work, more about it, and
 * the button of its fix where it offers one.
 */
import { Button } from '@numen/ui'
import type { PluginHealth } from './pluginInfo'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  item: PluginHealth
}>()

const emit = defineEmits<{
  (e: 'run-action', key: string): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onRunAction(): void {
  emit('run-action', props.item.key)
}
</script>

<template>
  <div
    class="plugin-health"
    :class="`plugin-health--${props.item.severity}`"
    :role="props.item.severity === 'error' ? 'alert' : 'status'"
    :aria-label="words.pluginState[props.item.severity]"
  >
    <span class="plugin-health__text">{{ props.item.text }}</span>
    <span v-if="props.item.detail" class="plugin-health__detail">{{ props.item.detail }}</span>
    <div v-if="props.item.actionLabel" class="plugin-health__actions">
      <Button size="small" @click="onRunAction">{{ props.item.actionLabel }}</Button>
    </div>
  </div>
</template>

<style scoped>
.plugin-health {
  --health-row-gap: 0.125rem;
  --health-spacing: 1.75rem;
  --health-pad-block: 0.5rem;
  --health-pad-inline: 0.75rem;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas:
    'text    actions'
    'detail  actions';
  column-gap: var(--numen-panel-gap);
  row-gap: var(--health-row-gap);
  align-items: start;
  margin-block-end: var(--health-spacing);
  padding: var(--health-pad-block) var(--health-pad-inline);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-tight);
  color: var(--numen-ink);
}

.plugin-health--error {
  border-color: var(--numen-alarm);
}

.plugin-health__text {
  grid-area: text;
  overflow-wrap: anywhere;
}

.plugin-health__detail {
  grid-area: detail;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}

.plugin-health__actions {
  grid-area: actions;
  display: flex;
}
</style>
