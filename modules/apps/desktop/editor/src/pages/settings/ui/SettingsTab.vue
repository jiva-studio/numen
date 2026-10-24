<script setup lang="ts">
/**
 * The settings tab: everything in numen.json a person can change, grouped by
 * the part of the application it governs.
 */
import { computed, ref, watch } from 'vue'
import type { SettingsTabState } from '../types'
import type { PluginInfo } from '@/entities/settings'
import SystemDetail from './SystemDetail.vue'
import { PluginsList, PluginDetail } from './plugins'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const initialPluginId = props.state.initialPluginId
const selectedPluginId = ref<string | null>(initialPluginId?.value ?? null)

watch(
  () => initialPluginId?.value,
  (newId) => {
    if (newId !== undefined) {
      selectedPluginId.value = newId
    }
  },
)

const plugins = computed<readonly PluginInfo[]>(() => props.state.installation.plugins?.value ?? [])
const selectedPlugin = computed<PluginInfo | undefined>(() =>
  plugins.value.find((one) => one.id === selectedPluginId.value),
)

/* -------------------------------- Handlers -------------------------------- */
function onSelectPlugin(pluginId: string): void {
  selectedPluginId.value = pluginId
  if (initialPluginId) {
    initialPluginId.value = pluginId
  }
}

function onBack(): void {
  selectedPluginId.value = null
  if (initialPluginId) {
    initialPluginId.value = null
  }
}
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <div class="settings__main">
        <SystemDetail v-if="selectedPluginId === 'system'" :state="props.state" @back="onBack" />
        <PluginDetail
          v-else-if="selectedPlugin"
          :state="props.state"
          :plugin="selectedPlugin"
          @back="onBack"
        />
        <PluginsList
          v-else
          :state="props.state"
          :plugins="plugins"
          @select-plugin="onSelectPlugin"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings {
  --settings-measure: 54rem;
  --settings-apart: 1.75rem;
  --settings-near: 0.375rem;
  --settings-value: 6rem;
  --settings-choice: 18rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.settings__page {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

.settings__main {
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}
</style>
