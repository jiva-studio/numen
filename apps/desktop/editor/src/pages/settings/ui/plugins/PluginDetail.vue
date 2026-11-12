<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with the declarative
 * settings drawn in its slot.
 */
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import type { PluginInfo } from '@/entities/settings'
import { usePluginDetail, usePluginSwitch } from '../../model/usePluginDetail'
import type { InstallationConnections, InstallationPlugin, InstallationSettings } from '../../types'
import PluginFields from './PluginFields.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  connections: InstallationConnections
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
const detailDeps = {
  getSettings: () => props.settings,
  getManager: () => props.manager,
  getPlugin: () => props.plugin,
}
const { managed, hasSettings, fileCount, deleteProblem, readmeText, deleteData } =
  usePluginDetail(detailDeps)
const { isEnabled, isPending, setEnabled, retry } = usePluginSwitch(detailDeps)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <PluginDetailView
    v-if="managed"
    :plugin="managed"
    :is-enabled="isEnabled"
    :is-pending="isPending"
    :readme="readmeText"
    :data-file-count="fileCount"
    :problem="deleteProblem"
    :can-install="false"
    :has-settings="hasSettings"
    @back="onBack"
    @toggle-enabled="setEnabled"
    @retry="retry"
    @delete-data="deleteData"
  >
    <template #settings>
      <PluginFields
        v-if="hasSettings"
        :settings="props.settings"
        :host="props.manager"
        :connections="props.connections"
        :plugin="props.plugin"
        @open-connections="emit('open-connections')"
      />
    </template>
  </PluginDetailView>
</template>
