/**
 * The enum and option-backed settings of a plugin: the options under the values
 * in force, their select choices with detail subtitles, the active value, and
 * the placement of secrets alongside the settings they belong to.
 */
import type { SelectChoice } from '@numen/ui'
import {
  formatValue,
  getOptions,
  getValue,
  isShown,
  type PluginInfo,
  type PluginSecret,
  type PluginSettingInfo,
  type PluginSettingOption,
  type ReadPluginSetting,
} from '@/entities/settings'

export interface EnumFields {
  getEnumOptions(setting: PluginSettingInfo): readonly PluginSettingOption[]
  getEnumChoices(setting: PluginSettingInfo): readonly SelectChoice[]
  getEnumValue(setting: PluginSettingInfo): string
  getShownSettings(): readonly PluginSettingInfo[]
  getTopSecrets(): readonly PluginSecret[]
  getSecretsAfter(settingKey: string): readonly PluginSecret[]
}

/** Converts plugin setting options into select choices, preserving option detail. */
export function toSelectChoices(options: readonly PluginSettingOption[]): readonly SelectChoice[] {
  return options.map((one) => ({
    id: one.value,
    text: one.title || one.value,
    ...(one.detail ? { detail: one.detail } : {}),
  }))
}

/** Sorts themes with built-in themes first and maps them into grouped select choices. */
export function toThemeChoices(
  themes: readonly { readonly name: string; readonly title: string; readonly isBuiltIn: boolean }[],
  shippedLabel: string,
  ownedLabel: string,
): readonly SelectChoice[] {
  return [...themes]
    .sort((one, other) => Number(other.isBuiltIn) - Number(one.isBuiltIn))
    .map((one) => ({
      id: one.name,
      text: one.title,
      group: one.isBuiltIn ? shippedLabel : ownedLabel,
    }))
}

/** Finds the setting key after which a secret should be placed, or null for the top. */
function findSecretAnchor(secret: PluginSecret, shownKeys: ReadonlySet<string>): string | null {
  if (secret.when?.setting && shownKeys.has(secret.when.setting)) {
    return secret.when.setting
  }
  if (shownKeys.has('provider')) return 'provider'
  if (shownKeys.has('baseUrl')) return 'baseUrl'
  return null
}

/** The plugin is asked for on every call, so a plugin that changes is followed. */
export function createEnumFields(
  getPlugin: () => PluginInfo,
  read: ReadPluginSetting,
): EnumFields {
  const getEnumOptions = (setting: PluginSettingInfo) => getOptions(getPlugin(), setting, read)
  const getEnumChoices = (setting: PluginSettingInfo) => toSelectChoices(getEnumOptions(setting))
  const getEnumValue = (setting: PluginSettingInfo) =>
    formatValue(getValue(getPlugin(), setting, read).value)

  const getShownSettings = () =>
    getPlugin().settings.filter((one) => isShown(getPlugin(), one.when, read))

  const getShownSecrets = () =>
    getPlugin().secrets.filter((one) => isShown(getPlugin(), one.when, read))

  const getShownKeys = () => new Set(getShownSettings().map((one) => one.key))

  const getTopSecrets = () => {
    const keys = getShownKeys()
    return getShownSecrets().filter((one) => findSecretAnchor(one, keys) === null)
  }

  const getSecretsAfter = (settingKey: string) => {
    const keys = getShownKeys()
    return getShownSecrets().filter((one) => findSecretAnchor(one, keys) === settingKey)
  }

  return {
    getEnumOptions,
    getEnumChoices,
    getEnumValue,
    getShownSettings,
    getTopSecrets,
    getSecretsAfter,
  }
}
