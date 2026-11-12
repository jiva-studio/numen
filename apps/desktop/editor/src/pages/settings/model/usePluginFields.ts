/**
 * The state of the declarative settings form of one plugin, in three parts the form calls one after
 * the other: the values in force with the text drafts, the enum choices, and the paths chosen
 * through the host's dialog.
 *
 * A path chosen for another plugin than the one shown when the dialog was opened is dropped.
 */
import { computed, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'
import { formatError } from '@/shared/api/errors'
import type { JsonValue } from '@/shared/api/json'
import { isChangedFromDefault } from '../lib/defaults'
import { createEnumFields, getValueInForce } from '../lib/enums'
import type { InstallationSettings, PluginFieldsHost } from '../types'
import { createPluginSettings } from './createPluginSettings'
import { appendPathEntry, removePathEntry, useTextDrafts } from './drafts'

export interface PluginFieldsDeps {
  readonly getSettings: () => InstallationSettings
  readonly getHost: () => PluginFieldsHost
  readonly getPlugin: () => PluginInfo
}

function hasManyPaths(setting: PluginSettingInfo): boolean {
  return setting.type === 'file' && setting.multiple === true
}

function createSettingValues(
  deps: PluginFieldsDeps,
  pluginSettings: ReturnType<typeof createPluginSettings>,
) {
  function getValue(key: string): JsonValue | undefined {
    const setting = deps.getPlugin().settings.find((one) => one.key === key)
    if (!setting) return pluginSettings.get(key)
    return getValueInForce(setting, pluginSettings.get, pluginSettings.isRequested)
  }

  return {
    getValue,
    getBooleanSetting: (key: string, fallback: JsonValue | undefined): boolean => {
      const val = getValue(key) ?? fallback
      return typeof val === 'boolean' ? val : Boolean(fallback)
    },
    getNumberSetting: (key: string, fallback: JsonValue | undefined): number => {
      const val = getValue(key) ?? fallback
      return typeof val === 'number' ? val : Number(fallback ?? 0)
    },
    getStringSetting: (key: string, fallback: JsonValue | undefined): string => {
      const val = getValue(key) ?? fallback
      return typeof val === 'string' ? val : String(fallback ?? '')
    },
  }
}

const createFieldSettings = (deps: PluginFieldsDeps) =>
  createPluginSettings(
    () => deps.getSettings(),
    () => deps.getPlugin().id,
  )

/** The values in force of the settings of the plugin, the text typed and not written yet, and the write of a value. */
export function usePluginFields(deps: PluginFieldsDeps) {
  const pluginSettings = createFieldSettings(deps)
  const values = createSettingValues(deps, pluginSettings)
  const { getDraft, setDraft, writeDraft } = useTextDrafts(pluginSettings.set)
  return {
    getValue: values.getValue,
    getBooleanSetting: values.getBooleanSetting,
    getNumberSetting: values.getNumberSetting,
    getStringSetting: values.getStringSetting,
    getDraft,
    setDraft,
    writeDraft,
    setSetting: pluginSettings.set,
  }
}

/** Whether each setting holds a value other than its default, and the write of the default back. */
export function createSettingDefaults(deps: PluginFieldsDeps) {
  const pluginSettings = createFieldSettings(deps)
  return {
    isChanged: (setting: PluginSettingInfo): boolean =>
      isChangedFromDefault(setting.default, pluginSettings.get(setting.key)),
    resetSetting: (setting: PluginSettingInfo): void => {
      if (setting.default !== undefined) pluginSettings.set(setting.key, setting.default)
    },
  }
}

/** The choices of the enum settings, the settings shown and the secrets placed among them. */
export function useEnumFields(deps: PluginFieldsDeps) {
  const pluginSettings = createFieldSettings(deps)
  const platform = computed(() => deps.getSettings().getPlatform())
  return createEnumFields(
    () => deps.getPlugin(),
    pluginSettings.get,
    pluginSettings.isRequested,
    () => platform.value,
  )
}

/** The paths chosen for the path settings of the plugin, and why one was not accepted. */
export function usePluginPaths(
  deps: PluginFieldsDeps,
  fields: Pick<ReturnType<typeof usePluginFields>, 'getStringSetting' | 'setSetting'>,
) {
  /** Why a path setting did not accept the path last chosen, by key. */
  const problems = ref<Record<string, string>>({})

  async function choosePathNow(setting: PluginSettingInfo): Promise<void> {
    const pluginId = deps.getPlugin().id
    const previous = fields.getStringSetting(setting.key, setting.default)
    const problem = await deps.getHost().choosePath(pluginId, setting.key)
    if (deps.getPlugin().id !== pluginId) return
    problems.value = { ...problems.value, [setting.key]: problem }
    if (problem === '' && previous !== '' && hasManyPaths(setting)) {
      fields.setSetting(
        setting.key,
        appendPathEntry(previous, fields.getStringSetting(setting.key, '')),
      )
    }
  }

  return {
    problems,
    hasManyPaths,
    choosePath: (setting: PluginSettingInfo): void => {
      const pluginId = deps.getPlugin().id
      runInBackground(choosePathNow(setting), (error) => {
        if (deps.getPlugin().id === pluginId) {
          problems.value = { ...problems.value, [setting.key]: formatError(error) }
        }
      })
    },
    clearPath: (setting: PluginSettingInfo, targetPath?: string): void => {
      problems.value = { ...problems.value, [setting.key]: '' }
      const current = fields.getStringSetting(setting.key, setting.default)
      fields.setSetting(setting.key, removePathEntry(current, targetPath))
    },
  }
}
