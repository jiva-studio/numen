/**
 * The settings as when conditions read them: `config.<dotted key>`, the application's settings under
 * `app` and a plugin's under its id, as VS Code reads `config.editor.tabCompletion`.
 */
import type { WhenSettings } from '@numen/plugin-loader'
import type { JsonValue } from '@/shared/api/json'
import type { PluginSettingInfo } from '@/entities/settings'

/** The owner whose settings the core declares. */
export const APP_OWNER = 'app'

/** An owner of settings and the settings it declares, with the values in force the core resolved. */
export interface SettingOwner {
  readonly id: string
  readonly name: string
  readonly settings: readonly PluginSettingInfo[]
}

const getConfigPath = (owner: string, key: string): readonly string[] =>
  owner === APP_OWNER ? [APP_OWNER, ...key.split('.')] : [...owner.split('.'), ...key.split('.')]

/** Every setting of the owners by its dotted key, nested. */
export function getSettingsConfig(owners: readonly SettingOwner[]): WhenSettings {
  const config: Record<string, JsonValue> = {}
  for (const owner of owners) {
    for (const one of owner.settings) {
      const path = getConfigPath(owner.id, one.key)
      let level = config
      const key = path.at(-1)
      if (key === undefined) continue
      for (const part of path.slice(0, -1)) {
        level[part] ??= {}
        level = level[part] as Record<string, JsonValue>
      }
      if (one.value !== undefined) level[key] = one.value
    }
  }
  return config
}

/**
 * The when condition of the command of a setting: the condition of the setting and those of the
 * settings it reads, each the value of the setting it names, as text, among the values it lists.
 */
export function getSettingWhen(owner: SettingOwner, setting: PluginSettingInfo): string {
  const parts: string[] = []
  let condition = setting.when
  const visited = new Set<string>()
  while (condition && !visited.has(condition.setting)) {
    visited.add(condition.setting)
    const values = condition.is.map((one) => `'${one.replaceAll("'", "\\'")}'`).join(', ')
    parts.push(
      `string(config.${getConfigPath(owner.id, condition.setting).join('.')}) in [${values}]`,
    )
    const named = condition.setting
    condition = owner.settings.find((one) => one.key === named)?.when
  }
  return parts.join(' && ')
}
