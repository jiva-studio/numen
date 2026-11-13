<script setup lang="ts">
/**
 * The plugin list of the settings tab under the toolbar of the first level: the plugin manager
 * drawn over the installation's plugins, the catalog and the enabled settings.
 */
import {
  PluginsList as PluginsListView,
  SettingsPage,
  type PluginHealth,
} from '@numen/plugin-manager'
import { usePluginsList } from '../../model/usePluginsList'
import type { InstallationSettings } from '../../types'
import type { CatalogEntry } from '@/entities/plugin'
import type { PluginInfo } from '@/entities/settings'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: Pick<InstallationSettings, 'getSetting' | 'write'>
  catalog: readonly CatalogEntry[]
  query: string
  plugins: readonly PluginInfo[]
  health?: Readonly<Record<string, readonly PluginHealth[]>>
}>()

const emit = defineEmits<{
  (e: 'select-plugin', pluginId: string): void
  (e: 'update:query', query: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const {
  searchQuery,
  managed,
  enabled,
  setEnabled: onTogglePlugin,
} = usePluginsList({
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
  <SettingsPage :path="[words.settings]">
    <div class="plugins-page">
      <PluginsListView
        v-model:query="searchQuery"
        :plugins="managed"
        :enabled="enabled"
        :health="props.health ?? {}"
        @select-plugin="onSelectPlugin"
        @toggle-enabled="onTogglePlugin"
      />
    </div>
  </SettingsPage>
</template>

<style scoped>
.plugins-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
</style>
