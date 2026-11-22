<script setup lang="ts">
/**
 * The declarative settings of a plugin: the card of the settings that name no group, and under it an
 * island with a heading for each group that has a setting to show.
 */
import { computed } from 'vue'
import { IslandCard } from '@numen/ui'
import { PluginSettingsCard } from '@numen/plugin-manager'
import type { InstallationSettings, MarkedSetting, PluginFieldsHost } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { useEnumFields } from '../../model/usePluginFields'
import { getSettingSections } from '../../lib/settingSections'
import PluginFields from './PluginFields.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  host: PluginFieldsHost
  plugin: PluginInfo
  marked: readonly string[]
  focused: MarkedSetting | null
}>()

/* --------------------------------- State ---------------------------------- */
const { getShownSettings } = useEnumFields({
  getSettings: () => props.settings,
  getHost: () => props.host,
  getPlugin: () => props.plugin,
})
const sections = computed(() => getSettingSections(getShownSettings(), props.plugin.settingGroups))
const hasViews = computed(() => (props.plugin.settingsViews ?? []).length > 0)
</script>

<template>
  <PluginSettingsCard v-if="sections.plain.length > 0 || hasViews">
    <PluginFields
      :settings="props.settings"
      :host="props.host"
      :plugin="props.plugin"
      :marked="props.marked"
      :focused="props.focused"
    />
  </PluginSettingsCard>
  <IslandCard
    v-for="island in sections.islands"
    :key="island.id"
    :title="island.title"
    :data-testid="`setting-group-${island.id}`"
  >
    <PluginFields
      :settings="props.settings"
      :host="props.host"
      :plugin="props.plugin"
      :marked="props.marked"
      :focused="props.focused"
      :group="island.id"
    />
  </IslandCard>
</template>
