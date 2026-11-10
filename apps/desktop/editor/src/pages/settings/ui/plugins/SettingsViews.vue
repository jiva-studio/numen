<script setup lang="ts">
/**
 * The views a plugin's frontend registered for its settings form, drawn at the top of the form
 * in the order the manifest declares them. A declared view without a registration draws nothing.
 */
import { computed } from 'vue'
import type { InstallationSettings, PluginFieldsHost } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { createPluginSettings } from '../../model/createPluginSettings'
import { getRegisteredViews, getSettingValues } from '../../lib/settingsViews'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  host: Pick<PluginFieldsHost, 'getSettingsViews'>
  plugin: PluginInfo
}>()

/* --------------------------------- State ---------------------------------- */
const pluginSettings = createPluginSettings(
  () => props.settings,
  () => props.plugin.id,
)
const views = computed(() =>
  getRegisteredViews(
    props.plugin.settingsViews ?? [],
    props.host.getSettingsViews(props.plugin.id),
  ),
)
const values = computed(() =>
  getSettingValues(props.plugin, pluginSettings.get, pluginSettings.isRequested),
)

/* -------------------------------- Handlers -------------------------------- */
function onSettingUpdate(key: string, value: unknown): void {
  pluginSettings.set(key, value)
}
</script>

<template>
  <component
    :is="view.component"
    v-for="view in views"
    :key="`${props.plugin.id}-view-${view.id}`"
    :plugin-id="props.plugin.id"
    :values="values"
    @update:setting="onSettingUpdate"
  />
</template>
