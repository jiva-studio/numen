<script setup lang="ts">
/**
 * Placeholder displayed when a tab's kind belongs to a disabled, stopped or missing plugin. The
 * plugin's name, feature and state come from the catalog; until the catalog answers, the pane shows
 * the loading state.
 */
import { computed, ref } from 'vue'
import { TriangleAlert } from '@lucide/vue'
import { Button, LoadingIndicator, runInBackground } from '@numen/ui'
import { formatErrorMessage } from '@numen/core-client'
import type { UnavailableState } from './unavailable'
import { getUnavailableView, type UnavailableView } from './unavailableView'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: UnavailableState
}>()

/* --------------------------------- State ---------------------------------- */
const isEnabling = ref(false)
const problem = ref('')
const isStarting = computed(() => {
  const lookup = props.state.lookup.value
  return (
    lookup.status === 'found' && props.state.services.isPluginStarting?.(lookup.entry.id) === true
  )
})
const view = computed(() => getUnavailableView(props.state.lookup.value, isEnabling.value))

/* -------------------------------- Handlers -------------------------------- */
function onAct(target: UnavailableView): void {
  const { pluginId, action } = target
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
  <div v-if="isStarting" class="tab-unavailable" role="status" data-starting>
    <div class="tab-unavailable__content">
      <LoadingIndicator />
      <p class="tab-unavailable__message">Starting the plugin for this tab…</p>
    </div>
  </div>
  <LoadingIndicator v-else-if="view === undefined" />
  <div v-else class="tab-unavailable" role="status">
    <div class="tab-unavailable__content">
      <TriangleAlert class="tab-unavailable__icon" aria-hidden="true" />
      <h3 class="tab-unavailable__title">{{ view.heading }}</h3>
      <p class="tab-unavailable__message">{{ view.message }}</p>
      <p v-if="problem" class="tab-unavailable__message" role="alert">{{ problem }}</p>
      <Button
        class="tab-unavailable__action"
        :disabled="isEnabling"
        :aria-busy="isEnabling"
        @click="onAct(view)"
      >
        {{ view.actionText }}
      </Button>
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
  inline-size: 2.5rem;
  block-size: 2.5rem;
  margin-block-end: 0.75rem;
  color: var(--numen-caution-fg);
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

.tab-unavailable__action {
  margin-block-start: 0.5rem;
}
</style>
