<script setup lang="ts">
/**
 * The settings of a plugin that hold connection rows, each as a card of its own under the card of
 * the other settings: a row with the title and what the setting is for, then the rows of connections. A setting
 * whose condition fails is not shown.
 */
import { PluginSettingsCard } from '@numen/plugin-manager'
import type { StoredRow } from '@/entities/connection'
import type { PluginInfo } from '@/entities/settings'
import { useEnumFields, usePluginFields } from '../../model/usePluginFields'
import type { InstallationConnections, InstallationSettings, PluginFieldsHost } from '../../types'
import SettingRow from '../setting-row/SettingRow.vue'
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
const deps = {
  getSettings: () => props.settings,
  getHost: () => props.host,
  getPlugin: () => props.plugin,
}
const { getValue, setSetting } = usePluginFields(deps)
const { getShownSettings } = useEnumFields(deps)

/* -------------------------------- Handlers -------------------------------- */
function onRowsChange(key: string, rows: readonly StoredRow[]): void {
  setSetting(
    key,
    rows.map((row) => ({ connection: row.connection, model: row.model, reasoning: row.reasoning })),
  )
}
</script>

<template>
  <template v-for="setting in getShownSettings()" :key="setting.key">
    <PluginSettingsCard v-if="setting.type === 'connections'">
      <SettingRow :at="`${props.plugin.id}-${setting.key}`" :name="setting.title" :detail="setting.description">
        <span />
      </SettingRow>
      <ConnectionRows
        :setting="setting"
        :value="getValue(setting.key)"
        :connections="props.connections"
        @change="(rows) => onRowsChange(setting.key, rows)"
        @open-connections="emit('open-connections')"
      />
    </PluginSettingsCard>
  </template>
</template>
