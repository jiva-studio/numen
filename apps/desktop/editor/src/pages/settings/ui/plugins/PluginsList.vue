<script setup lang="ts">
/**
 * The plugin list of the settings tab: the plugin manager drawn over the installation's plugins,
 * the catalog and the enabled settings.
 */
import { computed, ref, watch } from 'vue'
import { PluginsList as PluginsListView } from '@numen/plugin-manager'
import { createManagedFromEntries } from '../../lib/pluginListing'
import type { InstallationSettings } from '../../types'
import type { CatalogEntry } from '@/entities/plugin'
import { write, type PluginInfo } from '@/entities/settings'

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
const searchQuery = ref(props.query)

watch(
  () => props.query,
  (newQuery) => {
    searchQuery.value = newQuery
  },
)

watch(searchQuery, (newQuery) => {
  if (props.query !== newQuery) emit('update:query', newQuery)
})

const managed = computed(() => createManagedFromEntries(props.plugins, props.catalog))

const enabled = computed(() =>
  Object.fromEntries(props.plugins.map((one) => [one.id, isPluginEnabled(one.id)])),
)

/* -------------------------------- Handlers -------------------------------- */
function onSelectPlugin(pluginId: string): void {
  emit('select-plugin', pluginId)
}

function onTogglePlugin(pluginId: string, isEnabled: boolean): void {
  props.settings.write([{ at: [pluginId, 'enabled'], value: write(isEnabled) }])
}

/* -------------------------------- Helpers --------------------------------- */
function isPluginEnabled(pluginId: string): boolean {
  const val = props.settings.getSetting([pluginId, 'enabled'])
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
