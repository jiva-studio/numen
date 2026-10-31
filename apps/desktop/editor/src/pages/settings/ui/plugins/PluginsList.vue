<script setup lang="ts">
/**
 * The plugin list of the settings tab: the plugin manager drawn over the installation's plugins,
 * the catalog and the enabled settings.
 */
import { computed, ref, watch } from 'vue'
import { PluginsList as PluginsListView, createManagedPlugins } from '@numen/plugin-manager'
import type { SettingsTabState } from '../../types'
import { write, type PluginInfo } from '@/entities/settings'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  state: SettingsTabState
  plugins: readonly PluginInfo[]
}>()

const emit = defineEmits<{
  (e: 'select-plugin', pluginId: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const searchQuery = ref(props.state.initialSearchQuery?.value ?? '')

watch(
  () => props.state.initialSearchQuery?.value,
  (newQuery) => {
    if (newQuery !== undefined) {
      searchQuery.value = newQuery
    }
  },
)

watch(searchQuery, (newQuery) => {
  const initialQuery = props.state.initialSearchQuery
  if (initialQuery && initialQuery.value !== newQuery) {
    initialQuery.value = newQuery
  }
})

const managed = computed(() =>
  createManagedPlugins(props.plugins, props.state.installation.catalog?.value ?? []),
)

const enabled = computed(() =>
  Object.fromEntries(props.plugins.map((one) => [one.id, isPluginEnabled(one.id)])),
)

/* -------------------------------- Handlers -------------------------------- */
function onSelectPlugin(pluginId: string): void {
  emit('select-plugin', pluginId)
}

function onTogglePlugin(pluginId: string, isEnabled: boolean): void {
  props.state.installation.write([
    { at: ['plugins', pluginId, 'enabled'], value: write(isEnabled) },
  ])
}

/* -------------------------------- Helpers --------------------------------- */
function isPluginEnabled(pluginId: string): boolean {
  const val = props.state.installation.getSetting(['plugins', pluginId, 'enabled'])
  return typeof val === 'boolean' ? val : true
}
</script>

<template>
  <PluginsListView
    v-model:query="searchQuery"
    :plugins="managed"
    :enabled="enabled"
    @select-plugin="onSelectPlugin"
    @toggle-enabled="onTogglePlugin"
  />
</template>
