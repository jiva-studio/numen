<script setup lang="ts">
/**
 * The settings of a plugin that hold connection rows, each as a card of its own under the card of
 * the other settings: a row with the title and what the setting is for, then the rows of connections. A setting
 * whose condition fails is not shown. The choice to discover is kept in the plugin field `discover`, by
 * the key of each setting; a setting that declares discover discovers until it is turned off.
 */
import { writeRows, type StoredRow } from '@/entities/connection'
import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'
import type { JsonObject } from '@/shared/api/json'
import { useEnumFields, usePluginFields } from '../../model/usePluginFields'
import type { InstallationConnections, InstallationSettings, PluginFieldsHost } from '../../types'
import ConnectionRows from './connection-rows/ConnectionRows.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: InstallationSettings
  host: PluginFieldsHost
  connections: InstallationConnections
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
const DISCOVER_FIELD = 'discover'
const deps = {
  getSettings: () => props.settings,
  getHost: () => props.host,
  getPlugin: () => props.plugin,
}
const { getValue, setSetting } = usePluginFields(deps)
const { getShownSettings } = useEnumFields(deps)

/* -------------------------------- Handlers -------------------------------- */
function onRowsChange(key: string, rows: readonly StoredRow[]): void {
  setSetting(key, writeRows(rows))
}

function onDiscover(key: string, isOn: boolean): void {
  setSetting(DISCOVER_FIELD, { ...getDiscoverChoices(), [key]: isOn })
}

/* -------------------------------- Helpers --------------------------------- */
function getDiscoverChoices(): JsonObject {
  const choices = getValue(DISCOVER_FIELD)
  return typeof choices === 'object' && choices !== null && !Array.isArray(choices) ? choices : {}
}

function isDiscovering(setting: PluginSettingInfo): boolean {
  if (setting.discover !== true) return false
  const choice = getDiscoverChoices()[setting.key]
  return typeof choice === 'boolean' ? choice : true
}
</script>

<template>
  <template v-for="setting in getShownSettings()" :key="setting.key">
    <ConnectionRows
      v-if="setting.type === 'connections'"
      :setting="setting"
      :value="getValue(setting.key)"
      :connections="props.connections"
      :is-discovering="isDiscovering(setting)"
      @change="(rows) => onRowsChange(setting.key, rows)"
      @discover="(isOn) => onDiscover(setting.key, isOn)"
      @open-connections="emit('open-connections')"
    />
  </template>
</template>
