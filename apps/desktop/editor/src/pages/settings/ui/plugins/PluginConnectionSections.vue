<script setup lang="ts">
/**
 * The settings of a plugin that hold connection rows, each as a section of the plugin page under
 * the card of the other settings: the heading, what the setting is for, and the rows. A setting
 * whose condition fails is not shown.
 */
import type { StoredRow } from '@/entities/connection'
import type { PluginInfo } from '@/entities/settings'
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
    <section v-if="setting.type === 'connections'" class="connection-section">
      <h3 class="connection-section__title">{{ setting.title }}</h3>
      <p v-if="setting.description" class="connection-section__detail">
        {{ setting.description }}
      </p>
      <ConnectionRows
        :setting="setting"
        :value="getValue(setting.key)"
        :connections="props.connections"
        @change="(rows) => onRowsChange(setting.key, rows)"
        @open-connections="emit('open-connections')"
      />
    </section>
  </template>
</template>

<style scoped>
.connection-section {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-block-start: 1.25rem;
}

.connection-section__title {
  margin: 0;
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
  font-weight: 500;
}

.connection-section__detail {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
