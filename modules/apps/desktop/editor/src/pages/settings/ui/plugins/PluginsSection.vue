<script setup lang="ts">
/**
 * Master-detail plugins section: renders either the plugins registry list
 * or the dedicated settings screen for a selected plugin.
 */
import { computed, ref } from 'vue'
import PluginsList from './PluginsList.vue'
import PluginDetail from './PluginDetail.vue'
import type { SettingsTabState } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const selectedPluginId = ref<string | null>(null)

const plugins = computed<readonly PluginInfo[]>(() => props.state.installation.plugins?.value ?? [])

const selectedPlugin = computed<PluginInfo | undefined>(() =>
  plugins.value.find((one) => one.id === selectedPluginId.value),
)

/* -------------------------------- Handlers -------------------------------- */
function onSelectPlugin(pluginId: string): void {
  selectedPluginId.value = pluginId
}

function onBack(): void {
  selectedPluginId.value = null
}
</script>

<template>
  <section v-if="plugins.length > 0" class="settings__group" :aria-label="words.plugins">
    <h2 v-if="!selectedPlugin" class="settings__heading">{{ words.plugins }}</h2>

    <PluginDetail
      v-if="selectedPlugin"
      :state="props.state"
      :plugin="selectedPlugin"
      @back="onBack"
    />

    <PluginsList v-else :state="props.state" :plugins="plugins" @select-plugin="onSelectPlugin" />
  </section>
</template>

<style scoped>
.settings__group {
  max-inline-size: var(--settings-measure, 54rem);
  margin-block-end: var(--settings-apart, 1.75rem);
  margin-inline: auto;
}

.settings__heading {
  margin: 0 0 var(--settings-near, 0.375rem);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}
</style>
