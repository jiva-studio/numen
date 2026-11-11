<script setup lang="ts">
/**
 * The views a plugin's frontend registered for its settings form, drawn at the top of the form
 * in the order the manifest declares them. A declared view without a registration draws nothing.
 */
import type { InstallationSettings, PluginFieldsHost } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { useSettingsViews } from '../../model/useSettingsViews'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  host: Pick<PluginFieldsHost, 'getSettingsViews'>
  plugin: PluginInfo
}>()

/* --------------------------------- State ---------------------------------- */
const {
  views,
  values,
  setSetting: onSettingUpdate,
} = useSettingsViews({
  getSettings: () => props.settings,
  getHost: () => props.host,
  getPlugin: () => props.plugin,
})
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
