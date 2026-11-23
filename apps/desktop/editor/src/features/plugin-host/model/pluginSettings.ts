import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import {
  DEFAULT_PLUGIN_ENABLED,
  createPluginStatus,
  isPluginEnabledValue,
  type Disposable,
  type EditorSettings,
  type JsonValue,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { SettingEdit, SettingsStore } from '@/entities/settings'
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

const hasRowChange = (batch: readonly SettingEdit[]): boolean =>
  batch.length === 0 ||
  batch.some(({ at }) => ROW_KEYS.has(at[0] ?? '') || (at.length === 2 && at[1] === 'enabled'))

/**
 * Calls the handler after each batch of changes that can change the rows a plugin lists: a change of a
 * connection, of the connections detected on this machine, or of whether a plugin is enabled. An empty
 * batch is the first after the stream was lost, and may hold any of them.
 */
export const createRowChangeWatcher =
  (rest: SettingsStore) =>
  (handler: () => void): Disposable => ({
    dispose: rest.changes.subscribe((batch) => {
      if (hasRowChange(batch)) handler()
    }),
  })

/** The settings of a plugin as the plugin host reads them, and the handlers run when one changes. */
export function createPluginSettingsAccessor(
  rest: SettingsStore,
  reportError: (pluginId: string, error: unknown) => void,
) {
  const getSetting = (pluginId: string, key: string): JsonValue | undefined =>
    getPluginSetting(rest, pluginId, key)

  const watchSetting = (
    pluginId: string,
    key: string,
    handler: (value: JsonValue | undefined) => void | Promise<void>,
  ): Disposable => ({
    dispose: watch(
      () => getSetting(pluginId, key),
      (newVal) => {
        runInBackground(Promise.resolve(handler(newVal)), (error) => reportError(pluginId, error))
      },
    ),
  })

  const getFlag = (key: string, fallback: boolean): boolean => {
    const value = rest.read.getCommittedSetting(['app', 'editor', key])
    return typeof value === 'boolean' ? value : fallback
  }
  const hasLoaded = (): boolean => rest.state.app.value.length > 0

  const getEditorSettings = (): EditorSettings | undefined =>
    hasLoaded()
      ? { lineNumbers: getFlag('lineNumbers', false), softWrap: getFlag('softWrap', true) }
      : undefined

  const watchEditorSettings = (
    pluginId: string,
    handler: (settings: EditorSettings) => void | Promise<void>,
  ): Disposable => ({
    dispose: watch(
      [hasLoaded, () => getFlag('lineNumbers', false), () => getFlag('softWrap', true)],
      () => {
        const settings = getEditorSettings()
        if (!settings) return
        runInBackground(Promise.resolve(handler(settings)), (error) => reportError(pluginId, error))
      },
    ),
  })

  return { getSetting, watchSetting, getEditorSettings, watchEditorSettings }
}
