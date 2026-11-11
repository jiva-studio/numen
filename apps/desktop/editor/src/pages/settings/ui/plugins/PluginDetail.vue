<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with the declarative
 * settings and secrets drawn in its slot.
 */
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import type { PluginInfo } from '@/entities/settings'
import { usePluginDetail } from '../../model/usePluginDetail'
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
const {
  managed,
  isEnabled,
  isPending,
  hasSettings,
  fileCount,
  deleteProblem,
  readmeText,
  actions,
} = usePluginDetail({
  getSettings: () => props.settings,
  getManager: () => props.manager,
  getPlugin: () => props.plugin,
})

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onToggleEnabled(isOn: boolean): void {
  actions.setEnabled(isOn)
}

function onRetry(): void {
  actions.retry()
}

async function onDeleteData(): Promise<void> {
  await actions.deleteData()
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
    @toggle-enabled="onToggleEnabled"
    @retry="onRetry"
    @delete-data="onDeleteData"
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
