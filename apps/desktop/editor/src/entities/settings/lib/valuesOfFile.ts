import type { JsonValue } from '@/shared/api/json'
import type { PluginInfo, PluginSettingInfo } from './configuration'
import { getSettingAt } from './settingAt'

/** The first element of the path of a setting of the shell. */
const OWNER_APP = 'app'

const withValues = (
  settings: readonly PluginSettingInfo[],
  file: JsonValue,
  getPath: (key: string) => readonly string[],
): readonly PluginSettingInfo[] =>
  settings.map((one) => ({ ...one, value: getSettingAt(file, getPath(one.key)) }))

/**
 * The declared settings with the value each holds in `file`. A write answers with the file and not
 * with the declarations, so the values the declarations carry are taken from the file.
 */
export const getValuesOfFile = (
  file: JsonValue,
  app: readonly PluginSettingInfo[],
  plugins: readonly PluginInfo[],
): { readonly app: readonly PluginSettingInfo[]; readonly plugins: readonly PluginInfo[] } => ({
  app: withValues(app, file, (key) => [OWNER_APP, ...key.split('.')]),
  plugins: plugins.map((plugin) => ({
    ...plugin,
    settings: withValues(plugin.settings, file, (key) => [plugin.id, key]),
  })),
})
