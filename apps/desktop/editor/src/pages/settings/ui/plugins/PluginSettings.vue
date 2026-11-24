<script setup lang="ts">
/**
 * The settings page of a plugin. A tab strip shows when the plugin declares tabs. The default tab
 * "Settings" holds the views that name no tab, the card of the settings that name no group, and
 * under it an island with a heading for each group that has a setting to show; a declared tab holds
 * its views. A view draws its own cards, so none wraps the views.
 */
import { computed, ref } from 'vue'
import { IslandCard, SegmentedControl } from '@numen/ui'
import { PluginSettingsCard } from '@numen/plugin-manager'
import type { InstallationSettings, MarkedSetting, PluginFieldsHost } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { useEnumFields } from '../../model/usePluginFields'
import { getSettingSections } from '../../lib/settingSections'
import { hasRowSettings } from '../../lib/pluginSettings'
import { DEFAULT_TAB, listSettingsTabs } from '../../lib/settingsTabs'
import { WORDS as words } from '../../words'
import PluginFields from './PluginFields.vue'
import SettingsViews from './SettingsViews.vue'

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
const hasPlainRows = computed(() => hasRowSettings(sections.value.plain))
const tabs = computed(() => listSettingsTabs(props.plugin, words.settingsTab))
const chosen = ref(DEFAULT_TAB)
const current = computed(() =>
  tabs.value.some((one) => one.id === chosen.value) ? chosen.value : DEFAULT_TAB,
)
</script>

<template>
  <SegmentedControl
    v-if="tabs.length > 1"
    v-model="chosen"
    :choices="tabs"
    :aria-label="words.settingsTabs"
    class="plugin-settings__tabs"
    data-testid="plugin-settings-tabs"
  />
  <SettingsViews
    :settings="props.settings"
    :host="props.host"
    :plugin="props.plugin"
    :tab="current"
  />
  <template v-if="current === DEFAULT_TAB">
    <PluginSettingsCard v-if="hasPlainRows">
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
</template>

<style scoped>
.plugin-settings__tabs {
  align-self: flex-start;
}
</style>
