import { createSettingFields } from '@/entities/settings'
import type { InstallationSettings } from '../types'
import type { JsonValue } from '@/shared/api/json'

/** The settings of one plugin, read and written by the key its manifest declares, such as `agent.use`. */
export interface PluginSettings {
  readonly get: (key: string) => JsonValue | undefined
  readonly isRequested: (key: string) => boolean
  readonly set: (key: string, value: JsonValue) => void
}

const getPath = (pluginId: string, key: string): readonly string[] => [pluginId, ...key.split('.')]

/** The plugin and the settings are read on every call, so a page kept for another plugin reads the new one. */
export function createPluginSettings(
  getSettings: () => InstallationSettings,
  getPluginId: () => string,
): PluginSettings {
  const fields = createSettingFields(getSettings)
  return {
    get: (key) => getSettings().getSetting(getPath(getPluginId(), key)),
    isRequested: (key) => getSettings().isRequested(getPath(getPluginId(), key)),
    set: (key, value) => fields.setValue(getPath(getPluginId(), key), value),
  }
}
