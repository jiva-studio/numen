/**
 * The enum and option-backed settings of a plugin: the options under the values
 * in force, their select choices with detail subtitles, the active value, and
 * the placement of secrets alongside the settings they belong to.
 */
import type { SelectChoice } from '@numen/ui'
import type {
  PluginInfo,
  PluginSecret,
  PluginSettingInfo,
  PluginSettingOption,
} from '@/entities/settings'

/** The value a plugin's setting holds, or nothing where the file holds none. */
export type ReadPluginSetting = (key: string) => unknown

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

/** The labels the themes are grouped under, and the name of each plugin that contributes themes. */
export interface ThemeLabels {
  readonly shipped: string
  readonly owned: string
  /** The name a plugin is listed under, given its id. */
  readonly getPluginName: (pluginId: string) => string
}

/**
 * Maps themes into grouped select choices: the ones that ship first, then the
 * themes of each plugin under the plugin's name, then the person's own.
 */
export function toThemeChoices(
  themes: readonly {
    readonly name: string
    readonly title: string
    readonly isBuiltIn: boolean
    readonly pluginId: string
  }[],
  labels: ThemeLabels,
): readonly SelectChoice[] {
  const rank = (one: { readonly isBuiltIn: boolean; readonly pluginId: string }) =>
    one.isBuiltIn ? 0 : one.pluginId === '' ? 2 : 1
  return [...themes]
    .sort((one, other) => rank(one) - rank(other))
    .map((one) => ({
      id: one.name,
      text: one.title,
      group: one.isBuiltIn
        ? labels.shipped
        : one.pluginId === ''
          ? labels.owned
          : labels.getPluginName(one.pluginId),
    }))
}

/** The value as a condition compares it. */
const toText = (value: unknown): string => {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : JSON.stringify(value)
}

/**
 * The value in force of a setting: the one the core resolved, else what the file holds, else the
 * default. A value written and not yet read back is shown as requested.
 */
export function getValueInForce(
  setting: PluginSettingInfo,
  read: ReadPluginSetting,
  isPending: (key: string) => boolean,
): unknown {
  if (setting.value !== undefined && !isPending(setting.key)) return setting.value
  return read(setting.key) ?? setting.default
}

/** Whether the core found the condition of a setting to hold. Without an answer it is shown. */
export const isSettingShown = (setting: PluginSettingInfo): boolean => setting.isShown !== false

/** Whether the condition of a secret holds under the values in force of the settings it reads. */
export function isSecretShown(
  plugin: PluginInfo,
  secret: PluginSecret,
  read: ReadPluginSetting,
  isPending: (key: string) => boolean = () => false,
): boolean {
  if (!secret.when) return true
  const parent = plugin.settings.find((one) => one.key === secret.when?.setting)
  if (!parent || !isSettingShown(parent)) return false
  return secret.when.is.includes(toText(getValueInForce(parent, read, isPending)))
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

/** The plugin is requested on every call, so a changed plugin is picked up. */
export function createEnumFields(
  getPlugin: () => PluginInfo,
  read: ReadPluginSetting,
  isPending: (key: string) => boolean = () => false,
): EnumFields {
  const getEnumOptions = (setting: PluginSettingInfo) =>
    setting.activeOptions ?? setting.options ?? []
  const getEnumChoices = (setting: PluginSettingInfo) => toSelectChoices(getEnumOptions(setting))
  const getEnumValue = (setting: PluginSettingInfo) =>
    toText(getValueInForce(setting, read, isPending))

  const getShownSettings = () => getPlugin().settings.filter(isSettingShown)

  const getShownSecrets = () =>
    getPlugin().secrets.filter((one) => isSecretShown(getPlugin(), one, read, isPending))

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
