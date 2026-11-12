<script setup lang="ts">
/**
 * The card of a plugin the host suspended: why it stopped, a retry, and the error the host recorded.
 */
import { ref } from 'vue'
import { TriangleAlert } from '@lucide/vue'
import type { PluginHealth } from './pluginInfo'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  item: PluginHealth
}>()

const emit = defineEmits<{
  (e: 'retry'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isDetailShown = ref(false)

/* -------------------------------- Handlers -------------------------------- */
function onRetry(): void {
  emit('retry')
}

function onToggleDetail(): void {
  isDetailShown.value = !isDetailShown.value
}
</script>

<template>
  <div class="plugin-stopped" role="group" :aria-label="words.pluginState.stopped">
    <TriangleAlert class="plugin-stopped__icon" :size="20" :stroke-width="1.75" aria-hidden="true" />
    <div class="plugin-stopped__text">
      <span class="plugin-stopped__reason">{{ props.item.text }}</span>
      <code v-if="isDetailShown" class="plugin-stopped__detail">{{ props.item.detail }}</code>
      <span class="plugin-stopped__actions">
        <button type="button" class="plugin-stopped__action" @click="onRetry">
          {{ words.pluginRetry }}
        </button>
        <button
          v-if="props.item.detail"
          type="button"
          class="plugin-stopped__action"
          :aria-expanded="isDetailShown"
          @click="onToggleDetail"
        >
          {{ isDetailShown ? words.pluginHideDetails : words.pluginShowDetails }}
        </button>
      </span>
    </div>
  </div>
</template>

<style scoped>
.plugin-stopped {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  margin-block-end: 1.5rem;
  padding: var(--settings-card-pad, 1rem);
  min-block-size: var(--settings-row-min, 3.25rem);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
  color: var(--numen-ink);
}

.plugin-stopped__icon {
  flex: none;
  margin-block-start: 0.0625rem;
  color: var(--numen-alarm);
}

.plugin-stopped__text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 0.125rem;
  min-inline-size: 0;
}

.plugin-stopped__reason {
  overflow-wrap: anywhere;
}

.plugin-stopped__detail {
  margin-block-start: 0.375rem;
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.plugin-stopped__actions {
  display: flex;
  font-size: var(--numen-text-2);
  gap: 1rem;
  margin-block-start: 0.375rem;
}

.plugin-stopped__action {
  padding: 0;
  border: none;
  background: none;
  color: var(--numen-accent);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
}

.plugin-stopped__action:hover {
  text-decoration: underline;
  text-underline-offset: 0.2em;
}
</style>
