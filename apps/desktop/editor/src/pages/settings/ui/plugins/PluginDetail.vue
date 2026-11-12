<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with its health items and
 * the declarative settings drawn in its slot, the settings the fixes of the items name marked.
 */
import { computed } from 'vue'
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import type { PluginInfo } from '@/entities/settings'
import { usePluginDetail, usePluginSwitch } from '../../model/usePluginDetail'
import type { InstallationConnections, InstallationPlugin, InstallationSettings } from '../../types'
import PluginConnectionSections from './PluginConnectionSections.vue'
import PluginFields from './PluginFields.vue'
import { getManagedHealth, getMarkedSettings } from '../../lib/pluginHealth'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  connections: InstallationConnections
  plugin: PluginInfo
  markedSetting?: string
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
const items = computed(() =>
  props.manager.health.value.filter((item) => item.pluginId === props.plugin.id),
)
const health = computed(() =>
  getManagedHealth(items.value, props.plugin.id, props.manager.canRunCommand),
)
const isApple = computed(() => props.settings.getPlatform() === 'darwin')
const marked = computed(() => [
  ...getMarkedSettings(items.value, props.plugin.id),
  ...(props.markedSetting ? [props.markedSetting] : []),
])

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onRunAction(key: string): void {
  const action = items.value.find((item) => item.key === key)?.action
  if (action) props.manager.runCommand(action.command, action.values)
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
    :health="health"
    :is-apple="isApple"
    @back="onBack"
    @toggle-enabled="setEnabled"
    @retry="retry"
    @delete-data="deleteData"
    @run-action="onRunAction"
  >
    <template #settings>
      <PluginFields
        v-if="hasSettings"
        :settings="props.settings"
        :host="props.manager"
        :plugin="props.plugin"
        :marked="marked"
        :focused="props.markedSetting ?? ''"
      />
    </template>
    <template #sections>
      <PluginConnectionSections
        :settings="props.settings"
        :host="props.manager"
        :connections="props.connections"
        :plugin="props.plugin"
        @open-connections="emit('open-connections')"
      />
    </template>
  </PluginDetailView>
</template>
