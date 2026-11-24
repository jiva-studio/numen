<script setup lang="ts">
/**
 * The views of one tab that a plugin's frontend registered for its settings form, drawn one after
 * another in the order the manifest declares them. A declared view without a registration draws nothing.
 */
import type { InstallationSettings, PluginFieldsHost } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { useSettingsViews } from '../../model/useSettingsViews'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  host: Pick<PluginFieldsHost, 'getSettingsViews'>
  plugin: PluginInfo
  tab: string
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
  getTab: () => props.tab,
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
