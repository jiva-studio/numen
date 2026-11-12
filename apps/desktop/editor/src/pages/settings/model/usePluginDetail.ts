/**
 * The README and the vault data count of one plugin in the settings tab, read whenever the shown
 * plugin changes, and the action that deletes the plugin's vault data.
 *
 * The README and the file count are read for the plugin shown; an answer for another plugin is
 * dropped.
 */
import { computed, ref, watch, type Ref } from 'vue'
import { createRequestGuard, runInBackground } from '@numen/ui'
import type { PluginInfo } from '@/entities/settings'
import { formatError } from '@/shared/api/errors'
import type { JsonValue } from '@/shared/api/json'
import { createManagedFromEntries } from '../lib/pluginListing'
import { hasPluginSettings } from '../lib/pluginSettings'
import type { InstallationPlugin, InstallationSettings } from '../types'
import { createPluginSettings } from './createPluginSettings'

export interface PluginDetailDeps {
  readonly getSettings: () => InstallationSettings
  readonly getManager: () => InstallationPlugin
  readonly getPlugin: () => PluginInfo
}

function createDataDeletion(
  deps: PluginDetailDeps,
  deleteProblem: Ref<string>,
  readFileCount: (pluginId: string) => Promise<void>,
): () => void {
  async function deletePluginData(pluginId: string): Promise<void> {
    const problem = await deps.getManager().files.deletePluginData(pluginId)
    if (deps.getPlugin().id !== pluginId) return
    deleteProblem.value = problem
    await readFileCount(pluginId)
  }

  return () => {
    const pluginId = deps.getPlugin().id
    runInBackground(deletePluginData(pluginId), (error) => {
      if (deps.getPlugin().id === pluginId) deleteProblem.value = formatError(error)
    })
  }
}

/** Whether the plugin is enabled, whether a change of that is waiting to be confirmed, and the calls that change it or start it again. */
export function usePluginSwitch(deps: PluginDetailDeps) {
  const readEnabledValue = (): JsonValue | undefined =>
    deps.getSettings().getConfirmedSetting([deps.getPlugin().id, 'enabled'])
  const isEnabled = computed<boolean>(() => {
    const value = readEnabledValue()
    return typeof value === 'boolean' ? value : true
  })

  const isPending = computed<boolean>(() =>
    deps.getSettings().isPending([deps.getPlugin().id, 'enabled']),
  )

  const pluginSettings = createPluginSettings(
    () => deps.getSettings(),
    () => deps.getPlugin().id,
  )

  function setEnabled(isOn: boolean): void {
    pluginSettings.set('enabled', isOn)
  }

  function retry(): void {
    deps.getManager().retryPlugin(deps.getPlugin().id)
  }

  return { isEnabled, isPending, setEnabled, retry }
}

export function usePluginDetail(deps: PluginDetailDeps) {
  const managed = computed(
    () => createManagedFromEntries([deps.getPlugin()], deps.getManager().catalog.value)[0],
  )

  const hasSettings = computed(() => hasPluginSettings(deps.getPlugin()))
  /** How many files the plugin keeps in the shown vault; zero hides the delete action. */
  const fileCount = ref(0)
  const deleteProblem = ref('')
  const readmeText = ref('')
  const countReads = createRequestGuard()
  const readmeReads = createRequestGuard()

  async function readFileCount(pluginId: string): Promise<void> {
    const request = countReads.acquireRequest()
    let count: number
    try {
      count = await deps.getManager().files.getPluginData(pluginId)
    } catch {
      count = 0
    }
    if (request.isCurrent) fileCount.value = count
  }

  async function readReadme(pluginId: string): Promise<void> {
    const request = readmeReads.acquireRequest()
    const text = await deps.getManager().files.getReadme(pluginId)
    if (request.isCurrent) readmeText.value = text
  }

  watch(
    () => deps.getPlugin().id,
    (pluginId) => {
      deleteProblem.value = ''
      readmeText.value = ''
      fileCount.value = 0
      runInBackground(Promise.all([readFileCount(pluginId), readReadme(pluginId)]), (error) => {
        if (deps.getPlugin().id === pluginId) deleteProblem.value = formatError(error)
      })
    },
    { immediate: true },
  )

  const deleteData = createDataDeletion(deps, deleteProblem, readFileCount)

  return { managed, hasSettings, fileCount, deleteProblem, readmeText, deleteData }
}
