import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import {
  DEFAULT_PLUGIN_ENABLED,
  createPluginStatus,
  isPluginEnabledValue,
  type JsonValue,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { SettingEdit, SettingsStore } from '@/entities/settings'
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

const ROW_KEYS = new Set(['connections', 'detectedConnections'])

/**
 * Whether a batch can change the rows a plugin lists: a change of a connection, of the connections
 * detected on this machine, or of whether a plugin is enabled. An empty batch is the first after the
 * stream was lost, and may hold any of them.
 */
const hasRowChange = (batch: readonly SettingEdit[]): boolean =>
  batch.length === 0 ||
  batch.some(({ at }) => ROW_KEYS.has(at[0] ?? '') || (at.length === 2 && at[1] === 'enabled'))

/** Calls the handler after each batch of changes that can change the rows a plugin lists. */
export const createRowChangeSubscriber =
  (rest: SettingsStore) =>
  (handler: () => void): (() => void) =>
    rest.changes.subscribe((batch) => {
      if (hasRowChange(batch)) handler()
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
