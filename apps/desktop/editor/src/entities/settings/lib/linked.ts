/**
 * What a plugin's settings say of one another: the options a setting has under the value of
 * another, the value in force and where it comes from, and whether a setting or a secret is shown.
 * The host resolves the values it hands the plugin by the same rules.
 */
import type {
  PluginInfo,
  PluginSettingCondition,
  PluginSettingInfo,
  PluginSettingOption,
} from './configuration'

/** The value a plugin's setting holds, or nothing where the file holds none. */
export type ReadPluginSetting = (key: string) => unknown

/**
 * Where the value in force comes from: the person's choice, the setting's default, or the first
 * option the setting it follows offers.
 */
export type ValueOrigin = 'chosen' | 'default' | 'followed'

export interface PluginValue {
  readonly value: unknown
  readonly origin: ValueOrigin
}

/** The value as a condition and a list of options compare it. */
export function formatValue(value: unknown): string {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : JSON.stringify(value)
}

/** The setting a plugin declares under the key. */
export function findSetting(plugin: PluginInfo, key: string): PluginSettingInfo | undefined {
  return plugin.settings.find((one) => one.key === key)
}

/** The options of the setting under the values in force. */
export function getOptions(
  plugin: PluginInfo,
  setting: PluginSettingInfo,
  read: ReadPluginSetting,
): readonly PluginSettingOption[] {
  if (!setting.optionsBy) return setting.options ?? []
  const parent = findSetting(plugin, setting.optionsBy.setting)
  const parentValue = parent ? getValue(plugin, parent, read).value : undefined
  return setting.optionsBy.options[formatValue(parentValue)] ?? []
}

/** The value in force of the setting, and where it comes from. */
export function getValue(
  plugin: PluginInfo,
  setting: PluginSettingInfo,
  read: ReadPluginSetting,
): PluginValue {
  const stored = read(setting.key)
  if (setting.optionsBy) {
    if (setting.type === 'string' && typeof stored === 'string' && stored !== '') {
      return { value: stored, origin: 'chosen' }
    }
    const options = getOptions(plugin, setting, read)
    if (options.some((one) => one.value === formatValue(stored)) && stored !== undefined) {
      return { value: stored, origin: 'chosen' }
    }
    return { value: options[0]?.value ?? setting.default, origin: 'followed' }
  }
  if (stored === undefined || formatValue(stored) === formatValue(setting.default)) {
    return { value: stored ?? setting.default, origin: 'default' }
  }
  return { value: stored, origin: 'chosen' }
}

/** Whether a condition holds, and the conditions of the settings it reads. */
export function isShown(
  plugin: PluginInfo,
  when: PluginSettingCondition | undefined,
  read: ReadPluginSetting,
  depth = 0,
): boolean {
  if (!when) return true
  const parent = findSetting(plugin, when.setting)
  if (!parent || depth > plugin.settings.length) return false
  const value = formatValue(getValue(plugin, parent, read).value)
  return when.is.includes(value) && isShown(plugin, parent.when, read, depth + 1)
}
