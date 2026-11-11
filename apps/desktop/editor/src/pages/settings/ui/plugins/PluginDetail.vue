<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with the declarative
 * settings and secrets drawn in its slot.
 */
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import type { PluginInfo } from '@/entities/settings'
import { usePluginDetail, usePluginSwitch } from '../../model/usePluginDetail'
import type { InstallationPlugin, InstallationSettings, SecretChoices } from '../../types'
import PluginFields from './PluginFields.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  choices: SecretChoices
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'open-secrets'): void
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
        :choices="props.choices"
        :plugin="props.plugin"
        @open-secrets="emit('open-secrets')"
      />
    </template>
  </PluginDetailView>
</template>
