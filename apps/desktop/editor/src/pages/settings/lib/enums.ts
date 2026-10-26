/**
 * The enum settings of a plugin as a row draws them: the options under the values in force, the
 * value, what is said under the row, and where the value comes from.
 */
import {
  findSetting,
  formatValue,
  getOptions,
  getValue,
  type PluginInfo,
  type PluginSettingInfo,
  type PluginSettingOption,
  type ReadPluginSetting,
} from '@/entities/settings'

export interface EnumWords {
  readonly valueChosen: string
  readonly valueDefault: string
  valueFollowed(title: string): string
}

export interface EnumFields {
  getEnumOptions(setting: PluginSettingInfo): readonly PluginSettingOption[]
  getEnumValue(setting: PluginSettingInfo): string
  /** The detail of the option in force, else what the setting means. */
  getDetail(setting: PluginSettingInfo): string
  /** Where the value of an enum setting comes from. */
  getOrigin(setting: PluginSettingInfo): string
}

/** The plugin is asked for on every call, so a plugin that changes is followed. */
export function createEnumFields(
  getPlugin: () => PluginInfo,
  read: ReadPluginSetting,
  words: EnumWords,
): EnumFields {
  const getEnumOptions = (setting: PluginSettingInfo) => getOptions(getPlugin(), setting, read)
  const getEnumValue = (setting: PluginSettingInfo) =>
    formatValue(getValue(getPlugin(), setting, read).value)

  const getDetail = (setting: PluginSettingInfo): string => {
    if (setting.type !== 'enum') return setting.description
    const value = getEnumValue(setting)
    const option = getEnumOptions(setting).find((one) => one.value === value)
    return option?.detail || setting.description
  }

  const getOrigin = (setting: PluginSettingInfo): string => {
    const { origin } = getValue(getPlugin(), setting, read)
    if (origin === 'chosen') return words.valueChosen
    if (origin === 'default' || !setting.optionsBy) return words.valueDefault
    const followed = findSetting(getPlugin(), setting.optionsBy.setting)
    return words.valueFollowed(followed?.title ?? '')
  }

  return { getEnumOptions, getEnumValue, getDetail, getOrigin }
}
