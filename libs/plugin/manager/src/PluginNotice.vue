<script setup lang="ts">
/**
 * A health item of a plugin on its page: what does not work, the action that fixes it, and the
 * detail the host recorded.
 */
import { ref } from 'vue'
import { TriangleAlert } from '@lucide/vue'
import type { PluginHealth } from './pluginInfo'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  item: PluginHealth
  actionLabel: string
}>()

const emit = defineEmits<{
  (e: 'run-action'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isDetailExpanded = ref(false)

/* -------------------------------- Handlers -------------------------------- */
function onRunAction(): void {
  emit('run-action')
}

function onToggleDetail(): void {
  isDetailExpanded.value = !isDetailExpanded.value
}
</script>

<template>
  <div
    class="settings-card plugin-notice"
    :class="`plugin-notice--${props.item.severity}`"
    :role="props.item.severity === 'error' ? 'alert' : 'status'"
  >
    <TriangleAlert class="plugin-notice__icon" :size="20" :stroke-width="1.75" aria-hidden="true" />
    <div class="plugin-notice__text">
      <span class="plugin-notice__reason">{{ props.item.text }}</span>
      <code v-if="isDetailExpanded" class="plugin-notice__detail">{{ props.item.detail }}</code>
      <span class="plugin-notice__actions">
        <button
          v-if="props.actionLabel"
          type="button"
          class="plugin-notice__action"
          @click="onRunAction"
        >
          {{ props.actionLabel }}
        </button>
        <button
          v-if="props.item.detail"
          type="button"
          class="plugin-notice__action"
          :aria-expanded="isDetailExpanded"
          @click="onToggleDetail"
        >
          {{ isDetailExpanded ? words.pluginHideDetails : words.pluginShowDetails }}
        </button>
      </span>
    </div>
  </div>
</template>

<style scoped>
.plugin-notice {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: var(--settings-card-pad);
  color: var(--numen-ink);
}

.plugin-notice__icon {
  flex: none;
  margin-block-start: 0.0625rem;
  color: var(--numen-caution-fg);
}

.plugin-notice--error .plugin-notice__icon {
  color: var(--numen-alarm);
}

.plugin-notice__text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 0.125rem;
  min-inline-size: 0;
}

.plugin-notice__reason {
  overflow-wrap: anywhere;
}

.plugin-notice__detail {
  margin-block-start: 0.375rem;
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.plugin-notice__actions {
  display: flex;
  font-size: var(--numen-text-2);
  gap: 1rem;
  margin-block-start: 0.375rem;
}

.plugin-notice__action {
  padding: 0;
  border: none;
  background: none;
  color: var(--numen-accent);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
}

.plugin-notice__action:hover {
  text-decoration: underline;
  text-underline-offset: 0.2em;
}
</style>
