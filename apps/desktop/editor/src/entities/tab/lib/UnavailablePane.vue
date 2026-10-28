<script setup lang="ts">
/**
 * Placeholder displayed when a tab's kind belongs to a disabled or missing plugin.
 */
import { computed } from 'vue'
import {
  formatUnavailableFeature,
  getUnavailablePluginFilter,
  type UnavailableState,
} from './unavailable'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: UnavailableState
}>()

/* --------------------------------- State ---------------------------------- */
const featureName = computed(() => formatUnavailableFeature(props.state.kind))

/* -------------------------------- Handlers -------------------------------- */
function onOpenSettings(): void {
  globalThis.dispatchEvent(
    new CustomEvent('numen:open-settings', {
      detail: { filter: getUnavailablePluginFilter(props.state.kind) },
    }),
  )
}
</script>

<template>
  <div class="tab-unavailable" role="status">
    <div class="tab-unavailable__content">
      <div class="tab-unavailable__icon" aria-hidden="true">&#9888;</div>
      <h3 class="tab-unavailable__title">Plugin disabled</h3>
      <p class="tab-unavailable__message">
        The plugin for {{ featureName }} is disabled or missing.
      </p>
      <button type="button" class="tab-unavailable__button" @click="onOpenSettings">
        Open Settings
      </button>
    </div>
  </div>
</template>

<style scoped>
.tab-unavailable {
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 100%;
  block-size: 100%;
  padding: 2rem;
  box-sizing: border-box;
}

.tab-unavailable__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-inline-size: 28rem;
}

.tab-unavailable__icon {
  font-size: 2.5rem;
  margin-block-end: 0.75rem;
  line-height: 1;
  color: var(--numen-warning, #e6a700);
}

.tab-unavailable__title {
  margin: 0 0 0.5rem 0;
  font-size: var(--numen-text-4, 1.25rem);
  font-weight: 600;
  color: var(--numen-ink);
}

.tab-unavailable__message {
  margin: 0 0 0.75rem 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2, 0.95rem);
  line-height: 1.4;
}

.tab-unavailable__button {
  margin-block-start: 0.5rem;
  padding: 0.5rem 1.25rem;
  font-size: var(--numen-text-2, 0.875rem);
  font-family: inherit;
  font-weight: 500;
  border-radius: var(--numen-radius-small, 4px);
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  background: var(--numen-surface, #fff);
  color: var(--numen-ink);
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-unavailable__button:hover {
  background-color: var(--numen-surface-hover, rgba(125, 125, 125, 0.08));
}

.tab-unavailable__button:active {
  background-color: var(--numen-surface-active, rgba(125, 125, 125, 0.16));
}
</style>
