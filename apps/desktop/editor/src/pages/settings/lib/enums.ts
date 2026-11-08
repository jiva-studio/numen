/**
 * The enum and option-backed settings of a plugin: the options under the values
 * in force, their select choices with detail subtitles, the active value, and
 * the placement of secrets alongside the settings they belong to.
 */
import type { SelectChoice } from '@numen/ui'
import { createWhenReader } from '@numen/plugin-host'
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

/**
 * The value in force of a setting: the one the core resolved, else what the file holds, else the
 * default. A value requested and not yet in the declaration is shown as requested, so a control
 * shows the value the person set from the request until the declaration carries it.
 */
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

/** Whether the condition of a secret holds under the values in force of the settings it reads. */
export function isSecretShown(
  plugin: PluginInfo,
  secret: PluginSecret,
  read: ReadPluginSetting,
  isRequested: (key: string) => boolean = () => false,
): boolean {
  if (!secret.when) return true
  const parent = plugin.settings.find((one) => one.key === secret.when?.setting)
  if (!parent || !isSettingShown(parent)) return false
  return secret.when.is.includes(toText(getValueInForce(parent, read, isRequested)))
}

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
  isRequested: (key: string) => boolean = () => false,
  getPlatform: () => string = () => '',
): EnumFields {
  const whenReader = createWhenReader()
  /** An option with no condition, or one that cannot be read, is offered. */
  const isAvailable = (option: PluginSettingOption): boolean =>
    whenReader.getCondition(option.when)?.evaluate({ 'app.platform': getPlatform() }) ?? true
  const getEnumOptions = (setting: PluginSettingInfo) =>
    setting.activeOptions ?? setting.options ?? []
  const getEnumChoices = (setting: PluginSettingInfo) => toSelectChoices(getEnumOptions(setting), isAvailable)
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

  const getShownSecrets = () =>
    getPlugin().secrets.filter((one) => isSecretShown(getPlugin(), one, read, isRequested))

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
