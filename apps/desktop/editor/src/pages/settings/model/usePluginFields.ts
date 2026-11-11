/**
 * The state and actions of the declarative settings form of one plugin: the values in force, the
 * enum choices, the text drafts and the paths chosen through the host's dialog.
 *
 * A path chosen for another plugin than the one shown when the dialog was opened is dropped.
 */
import { ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'
import { formatError } from '@/shared/api/errors'
import type { JsonValue } from '@/shared/api/json'
import { createEnumFields, getValueInForce } from '../lib/enums'
import type { InstallationSettings, PluginFieldsHost } from '../types'
import { createPluginSettings } from './createPluginSettings'
import { appendPathEntry, createTextDrafts, removePathEntry } from './drafts'

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
    getBooleanSetting(key: string, fallback: JsonValue | undefined): boolean {
      const val = getValue(key) ?? fallback
      return typeof val === 'boolean' ? val : Boolean(fallback)
    },
    getNumberSetting(key: string, fallback: JsonValue | undefined): number {
      const val = getValue(key) ?? fallback
      return typeof val === 'number' ? val : Number(fallback ?? 0)
    },
    getStringSetting(key: string, fallback: JsonValue | undefined): string {
      const val = getValue(key) ?? fallback
      return typeof val === 'string' ? val : String(fallback ?? '')
    },
  }
}

export function usePluginFields(deps: PluginFieldsDeps) {
  const pluginSettings = createPluginSettings(
    () => deps.getSettings(),
    () => deps.getPlugin().id,
  )
  const values = createSettingValues(deps, pluginSettings)
  const enums = createEnumFields(
    () => deps.getPlugin(),
    pluginSettings.get,
    pluginSettings.isRequested,
    () => deps.getSettings().getPlatform(),
  )

  /** Why a path setting did not accept the path last chosen, by key. */
  const problems = ref<Record<string, string>>({})
  const drafts = createTextDrafts(pluginSettings.set)

  async function choosePathNow(setting: PluginSettingInfo): Promise<void> {
    const pluginId = deps.getPlugin().id
    const previous = values.getStringSetting(setting.key, setting.default)
    const problem = await deps.getHost().choosePath(pluginId, setting.key)
    if (deps.getPlugin().id !== pluginId) return
    problems.value = { ...problems.value, [setting.key]: problem }
    if (problem === '' && previous !== '' && hasManyPaths(setting)) {
      pluginSettings.set(
        setting.key,
        appendPathEntry(previous, values.getStringSetting(setting.key, '')),
      )
    }
  }

  const paths = {
    problems,
    hasMany: hasManyPaths,
    choose(setting: PluginSettingInfo): void {
      const pluginId = deps.getPlugin().id
      runInBackground(choosePathNow(setting), (error) => {
        if (deps.getPlugin().id === pluginId) {
          problems.value = { ...problems.value, [setting.key]: formatError(error) }
        }
      })
    },
    clear(setting: PluginSettingInfo, targetPath?: string): void {
      problems.value = { ...problems.value, [setting.key]: '' }
      const current = values.getStringSetting(setting.key, setting.default)
      pluginSettings.set(setting.key, removePathEntry(current, targetPath))
    },
  }

  return { enums, values, drafts, paths, setSetting: pluginSettings.set }
}
