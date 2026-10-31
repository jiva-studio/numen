<script setup lang="ts">
/**
 * Placeholder displayed when a tab's kind belongs to a disabled, stopped or missing plugin. The
 * plugin's name, feature and state come from the catalog.
 */
import { computed, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { UnavailableState } from './unavailable'
import { describeUnavailable } from './unavailableView'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: UnavailableState
}>()

/* --------------------------------- State ---------------------------------- */
const isEnabling = ref(false)
const problem = ref('')
const view = computed(() => describeUnavailable(props.state.lookup.value, isEnabling.value))

/* -------------------------------- Handlers -------------------------------- */
function onAct(): void {
  const { pluginId, action } = view.value
  const { services } = props.state
  if (action === 'enable' && pluginId !== undefined && services.enablePlugin) {
    isEnabling.value = true
    problem.value = ''
    runInBackground(services.enablePlugin(pluginId), (error) => {
      isEnabling.value = false
      problem.value = formatErrorMessage(error)
    })
    return
  }
  services.openPlugin?.(pluginId)
}
</script>

<template>
  <div class="tab-unavailable" role="status">
    <div class="tab-unavailable__content">
      <div class="tab-unavailable__icon" aria-hidden="true">&#9888;</div>
      <h3 class="tab-unavailable__title">{{ view.heading }}</h3>
      <p class="tab-unavailable__message">{{ view.message }}</p>
      <p v-if="problem" class="tab-unavailable__message" role="alert">{{ problem }}</p>
      <button type="button" class="tab-unavailable__button" :disabled="isEnabling" @click="onAct">
        {{ view.actionText }}
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
  color: var(--numen-warning);
}

.tab-unavailable__title {
  margin: 0 0 0.5rem 0;
  font-size: var(--numen-text-4);
  font-weight: 600;
  color: var(--numen-ink);
}

.tab-unavailable__message {
  margin: 0 0 0.75rem 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  line-height: 1.4;
}

.tab-unavailable__button {
  margin-block-start: 0.5rem;
  padding: 0.5rem 1.25rem;
  font-size: var(--numen-text-2);
  font-family: inherit;
  font-weight: 500;
  border-radius: var(--numen-radius-small);
  border: var(--numen-stroke) solid var(--numen-rule);
  background: var(--numen-surface);
  color: var(--numen-ink);
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-unavailable__button:hover {
  background-color: var(--numen-surface-hover);
}

.tab-unavailable__button:active {
  background-color: var(--numen-surface-active);
}
</style>
