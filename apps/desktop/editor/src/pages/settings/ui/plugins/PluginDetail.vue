<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with the README and the
 * data of the plugin read here, and the declarative settings and secrets drawn in its slot.
 */
import { computed, ref, watch } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { PluginDetail as PluginDetailView, createManagedPlugins } from '@numen/plugin-manager'
import { runInBackground } from '@numen/ui'
import PluginFields from './PluginFields.vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, write, type PluginInfo } from '@/entities/settings'
import { hasPluginSettings } from '../../lib/pluginSettings'
import { fetchPluginReadme } from '../../lib/readme'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const managed = computed(() => {
  const [one] = createManagedPlugins([props.plugin], installation.value.catalog?.value ?? [])
  return one
})

const isEnabled = computed<boolean>(() => {
  const val = installation.value.getSetting(['plugins', props.plugin.id, 'enabled'])
  return typeof val === 'boolean' ? val : true
})

const read = (key: string) => installation.value.getSetting(['plugins', props.plugin.id, key])
const hasSettings = computed(() => hasPluginSettings(props.plugin, read))
/** How many files the plugin keeps in the shown vault; zero hides the delete action. */
const fileCount = ref(0)
const deleteProblem = ref('')
const readmeText = ref('')

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onToggleEnabled(isOn: boolean): void {
  fields.setValue(['plugins', props.plugin.id, 'enabled'], isOn)
}

function onRetry(): void {
  installation.value.retryPlugin(props.plugin.id)
}

function onConsentChange(isAllowed: boolean): void {
  installation.value.write([
    { at: ['plugins', props.plugin.id, 'aiConsent'], value: write(isAllowed) },
  ])
}

async function onDeleteData(): Promise<void> {
  deleteProblem.value = (await installation.value.deletePluginData?.(props.plugin.id)) ?? ''
  await readFileCount()
}

/* -------------------------------- Helpers --------------------------------- */
async function readFileCount(): Promise<void> {
  try {
    fileCount.value = (await installation.value.getPluginData?.(props.plugin.id)) ?? 0
  } catch {
    fileCount.value = 0
  }
}

async function readReadme(pluginId: string): Promise<void> {
  const text = await fetchPluginReadme(pluginId)
  if (text !== '' && props.plugin.id === pluginId) readmeText.value = text
}

watch(
  () => props.plugin.id,
  (pluginId) => {
    deleteProblem.value = ''
    readmeText.value = ''
    runInBackground(
      Promise.all([readFileCount(), readReadme(pluginId)]),
      (error) => (deleteProblem.value = formatErrorMessage(error)),
    )
  },
  { immediate: true },
)
</script>

<template>
  <PluginDetailView
    v-if="managed"
    :plugin="managed"
    :is-enabled="isEnabled"
    :readme="readmeText"
    :data-file-count="fileCount"
    :problem="deleteProblem"
    :can-install="false"
    :has-settings="hasSettings"
    @back="onBack"
    @toggle-enabled="onToggleEnabled"
    @retry="onRetry"
    @consent-change="onConsentChange"
    @delete-data="onDeleteData"
  >
    <template #settings>
      <PluginFields v-if="hasSettings" :state="props.state" :plugin="props.plugin" />
    </template>
  </PluginDetailView>
</template>
