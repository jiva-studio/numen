/**
 * The enum and option-backed settings of a plugin: the options under the values
 * in force, their select choices with detail subtitles, the active value, and
 */
import type { SelectChoice } from '@numen/ui'
import { createWhenReader } from '@numen/plugin-host'
import type { PluginInfo, PluginSettingInfo, PluginSettingOption } from '@/entities/settings'

/** The value a plugin's setting holds, or nothing where the file holds none. */
export type ReadPluginSetting = (key: string) => unknown

export interface EnumFields {
  getEnumOptions(this: void, setting: PluginSettingInfo): readonly PluginSettingOption[]
  getEnumChoices(this: void, setting: PluginSettingInfo): readonly SelectChoice[]
  getEnumValue(this: void, setting: PluginSettingInfo): string
  getShownSettings(this: void): readonly PluginSettingInfo[]
}

/**
 * Converts plugin setting options into select choices, preserving option detail. An option that
 * is not offered is a disabled choice.
 */
export function toSelectChoices(
  options: readonly PluginSettingOption[],
  isAvailable: (option: PluginSettingOption) => boolean = () => true,
): readonly SelectChoice[] {
  return options.map((one) => ({
    id: one.value,
    text: one.title || one.value,
    ...(one.detail ? { detail: one.detail } : {}),
    ...(isAvailable(one) ? {} : { isDisabled: true }),
  }))
}

const toText = (value: unknown): string => {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : JSON.stringify(value)
}

/** Returns the active value of a setting, falling back to the stored or default value. */
export function getValueInForce(
  setting: PluginSettingInfo,
  read: ReadPluginSetting,
  isRequested: (key: string) => boolean,
): unknown {
  if (setting.value !== undefined && !isRequested(setting.key)) return setting.value
  return read(setting.key) ?? setting.default
}

/** Whether the core found the condition of a setting to hold. Without an answer it is shown. */
export const isSettingShown = (setting: PluginSettingInfo): boolean => setting.isShown !== false

/** The plugin is requested on every call, so a changed plugin is picked up. */
export function createEnumFields(
  getPlugin: () => PluginInfo,
  read: ReadPluginSetting,
  isRequested: (key: string) => boolean = () => false,
  getPlatform: () => string = () => '',
): EnumFields {
  const whenReader = createWhenReader()
  /** An option with no condition, or one that cannot be read, is offered. */
  const isAvailable = (option: PluginSettingOption): boolean =>
    whenReader.getCondition(option.when)?.evaluate({ 'app.platform': getPlatform() }) ?? true
  const getEnumOptions = (setting: PluginSettingInfo) =>
    setting.activeOptions ?? setting.options ?? []
  const getEnumChoices = (setting: PluginSettingInfo) =>
    toSelectChoices(getEnumOptions(setting), isAvailable)
  /**
   * The value the select shows: the value in force, or the first offered option when the value in
   * force is an option this platform does not offer. Showing it writes nothing.
   */
  const getEnumValue = (setting: PluginSettingInfo) => {
    const value = toText(getValueInForce(setting, read, isRequested))
    const options = getEnumOptions(setting)
    const current = options.find((one) => one.value === value)
    if (!current || isAvailable(current)) return value
    return options.find(isAvailable)?.value ?? value
  }

  const getShownSettings = () => getPlugin().settings.filter(isSettingShown)

  return {
    getEnumOptions,
    getEnumChoices,
    getEnumValue,
    getShownSettings,
  }
}
