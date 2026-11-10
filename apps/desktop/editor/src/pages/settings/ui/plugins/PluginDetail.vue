<script setup lang="ts">
import type { JsonValue } from '@/shared/api/json'
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with the README and the
 * data of the plugin read here, and the declarative settings and secrets drawn in its slot.
 *
 * The README and the file count are read for the plugin shown; an answer for another plugin is
 * dropped.
 */
import { computed, ref, watch } from 'vue'
import { formatError } from '@/shared/api/errors'
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import { createManagedFromEntries } from '../../lib/pluginListing'
import { createRequestGuard, runInBackground } from '@numen/ui'
import PluginFields from './PluginFields.vue'
import type { InstallationPlugin, InstallationSettings, SecretChoices } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { createPluginSettings } from '../../model/createPluginSettings'
import { hasPluginSettings } from '../../lib/pluginSettings'

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
const pluginSettings = createPluginSettings(
  () => props.settings,
  () => props.plugin.id,
)

const read = (key: string) => props.settings.getSetting([props.plugin.id, key])

const managed = computed(
  () => createManagedFromEntries([props.plugin], props.manager.catalog.value)[0],
)

const readEnabled = (): JsonValue | undefined => props.settings.getConfirmedSetting([props.plugin.id, 'enabled'])
const isEnabled = computed<boolean>(() => {
  const value = readEnabled()
  return typeof value === 'boolean' ? value : true
})

const isPending = computed<boolean>(() => props.settings.isPending([props.plugin.id, 'enabled']))

const hasSettings = computed(() => hasPluginSettings(props.plugin, read))
/** How many files the plugin keeps in the shown vault; zero hides the delete action. */
const fileCount = ref(0)
const deleteProblem = ref('')
const readmeText = ref('')
const countReads = createRequestGuard()
const readmeReads = createRequestGuard()

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onToggleEnabled(isOn: boolean): void {
  pluginSettings.set('enabled', isOn)
}

function onRetry(): void {
  props.manager.retryPlugin(props.plugin.id)
}

async function onDeleteData(): Promise<void> {
  const pluginId = props.plugin.id
  const problem = await props.manager.files.deletePluginData(pluginId)
  if (props.plugin.id !== pluginId) return
  deleteProblem.value = problem
  await readFileCount(pluginId)
}

/* -------------------------------- Helpers --------------------------------- */
async function readFileCount(pluginId: string): Promise<void> {
  const request = countReads.acquireRequest()
  let count: number
  try {
    count = await props.manager.files.getPluginData(pluginId)
  } catch {
    count = 0
  }
  if (request.isCurrent) fileCount.value = count
}

async function readReadme(pluginId: string): Promise<void> {
  const request = readmeReads.acquireRequest()
  const text = await props.manager.files.getReadme(pluginId)
  if (request.isCurrent) readmeText.value = text
}

watch(
  () => props.plugin.id,
  (pluginId) => {
    deleteProblem.value = ''
    readmeText.value = ''
    fileCount.value = 0
    runInBackground(Promise.all([readFileCount(pluginId), readReadme(pluginId)]), (error) => {
      if (props.plugin.id === pluginId) deleteProblem.value = formatError(error)
    })
  },
  { immediate: true },
)
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
