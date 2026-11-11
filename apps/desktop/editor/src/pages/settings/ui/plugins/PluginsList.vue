<script setup lang="ts">
/**
 * The plugin list of the settings tab: the plugin manager drawn over the installation's plugins,
 * the catalog and the enabled settings.
 */
import { PluginsList as PluginsListView } from '@numen/plugin-manager'
import { usePluginsList } from '../../model/usePluginsList'
import type { InstallationSettings } from '../../types'
import type { CatalogEntry } from '@/entities/plugin'
import type { PluginInfo } from '@/entities/settings'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: Pick<InstallationSettings, 'getSetting' | 'write'>
  catalog: readonly CatalogEntry[]
  query: string
  plugins: readonly PluginInfo[]
}>()

const emit = defineEmits<{
  (e: 'select-plugin', pluginId: string): void
  (e: 'update:query', query: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const { searchQuery, managed, enabled, setEnabled: onTogglePlugin } = usePluginsList({
  getSettings: () => props.settings,
  getCatalog: () => props.catalog,
  getQuery: () => props.query,
  getPlugins: () => props.plugins,
  updateQuery: (query) => emit('update:query', query),
})

/* -------------------------------- Handlers -------------------------------- */
function onSelectPlugin(pluginId: string): void {
  emit('select-plugin', pluginId)
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
