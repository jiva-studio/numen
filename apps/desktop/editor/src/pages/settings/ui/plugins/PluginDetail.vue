<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with the README and the
 * data of the plugin read here, and the declarative settings drawn in its slot.
 *
 * The README and the file count are read for the plugin shown; an answer for another plugin is
 * dropped.
 */
import { computed, ref, watch } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import { createManagedFromEntries } from '@/shared/pluginListing'
import { createRequestGuard, runInBackground } from '@numen/ui'
import PluginFields from './PluginFields.vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, type PluginInfo } from '@/entities/settings'
import { hasPluginSettings } from '../../lib/pluginSettings'
import { fetchPluginReadme } from '../../lib/readme'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const managed = computed(
  () => createManagedFromEntries([props.plugin], installation.value.catalog?.value ?? [])[0],
)

const readEnabled = (): unknown =>
  (installation.value.getConfirmedSetting ?? installation.value.getSetting)([
    props.plugin.id,
    'enabled',
  ])
const isEnabled = computed<boolean>(() => {
  const value = readEnabled()
  return typeof value === 'boolean' ? value : true
})

const isPending = computed<boolean>(
  () => installation.value.isPending?.([props.plugin.id, 'enabled']) ?? false,
)

const hasSettings = computed(() => hasPluginSettings(props.plugin))
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
  fields.setValue([props.plugin.id, 'enabled'], isOn)
}

function onRetry(): void {
  installation.value.retryPlugin(props.plugin.id)
}

async function onDeleteData(): Promise<void> {
  const pluginId = props.plugin.id
  const problem = (await installation.value.deletePluginData?.(pluginId)) ?? ''
  if (props.plugin.id !== pluginId) return
  deleteProblem.value = problem
  await readFileCount(pluginId)
}

/* -------------------------------- Helpers --------------------------------- */
async function readFileCount(pluginId: string): Promise<void> {
  const request = countReads.acquireRequest()
  let count: number
  try {
    count = (await installation.value.getPluginData?.(pluginId)) ?? 0
  } catch {
    count = 0
  }
  if (request.isCurrent) fileCount.value = count
}

async function readReadme(pluginId: string): Promise<void> {
  const request = readmeReads.acquireRequest()
  const text = await fetchPluginReadme(pluginId)
  if (request.isCurrent) readmeText.value = text
}

watch(
  () => props.plugin.id,
  (pluginId) => {
    deleteProblem.value = ''
    readmeText.value = ''
    fileCount.value = 0
    runInBackground(Promise.all([readFileCount(pluginId), readReadme(pluginId)]), (error) => {
      if (props.plugin.id === pluginId) deleteProblem.value = formatErrorMessage(error)
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
        :state="props.state"
        :plugin="props.plugin"
        @open-connections="emit('open-connections')"
      />
    </template>
  </PluginDetailView>
</template>
