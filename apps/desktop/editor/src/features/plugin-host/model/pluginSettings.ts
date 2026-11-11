import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import {
  DEFAULT_PLUGIN_ENABLED,
  createPluginStatus,
  isPluginEnabledValue,
  type JsonValue,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { SettingsStore } from '@/entities/settings'
import type { Caught } from '@/shared/api/errors'

const getPluginSetting = (
  rest: SettingsStore,
  pluginId: string,
  key: string,
): JsonValue | undefined => {
  const value = rest.read.getCommittedSetting([pluginId, key])
  if (value !== undefined) return value
  return key === 'enabled' ? DEFAULT_PLUGIN_ENABLED : undefined
}

/** Where the plugins stand, as the settings say which are enabled. */
export const createWindowPluginStatus = (rest: SettingsStore): PluginStatus =>
  createPluginStatus({
    isEnabled: (id) => isPluginEnabledValue(getPluginSetting(rest, id, 'enabled')),
  })

/** The settings of a plugin as the plugin host reads them, and the handlers run when one changes. */
export function createPluginSettingsAccessor(
  rest: SettingsStore,
  reportError: (pluginId: string, error: Caught) => void,
) {
  const getSetting = (pluginId: string, key: string): JsonValue | undefined =>
    getPluginSetting(rest, pluginId, key)

  const onSettingChange = (
    pluginId: string,
    key: string,
    handler: (value: JsonValue | undefined) => void | Promise<void>,
  ): (() => void) =>
    watch(
      () => getSetting(pluginId, key),
      (newVal) => {
        runInBackground(Promise.resolve(handler(newVal)), (error) => reportError(pluginId, error))
      },
    )

  return { getSetting, onSettingChange }
}
