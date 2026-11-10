/**
 * The enum and option-backed settings of a plugin: the options under the values
 * in force, their select choices with detail subtitles, the active value, and
 * the placement of secrets alongside the settings they belong to.
 */
import type { SelectChoice } from '../types'
import { createWhenReader } from '@numen/plugin-loader'
import type {
  PluginInfo,
  SecretDeclaration,
  PluginSettingInfo,
  PluginSettingOption,
} from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'

/** The value a plugin's setting holds, or nothing where the file holds none. */
export type ReadPluginSetting = (key: string) => JsonValue | undefined

export interface EnumFields {
  getEnumOptions(this: void, setting: PluginSettingInfo): readonly PluginSettingOption[]
  getEnumChoices(this: void, setting: PluginSettingInfo): readonly SelectChoice[]
  getEnumValue(this: void, setting: PluginSettingInfo): string
  getShownSettings(this: void): readonly PluginSettingInfo[]
  getTopSecrets(this: void): readonly SecretDeclaration[]
  getSecretsAfter(this: void, settingKey: string): readonly SecretDeclaration[]
}

function toSelectChoices(
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

const toText = (value: JsonValue | undefined): string => {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : JSON.stringify(value)
}

/** Returns the active value of a setting, falling back to the stored or default value. */
export function getValueInForce(
  setting: PluginSettingInfo,
  read: ReadPluginSetting,
  isRequested: (key: string) => boolean,
): JsonValue | undefined {
  if (setting.value !== undefined && !isRequested(setting.key)) return setting.value
  return read(setting.key) ?? setting.default
}

const isSettingShown = (setting: PluginSettingInfo): boolean => setting.isShown !== false

/** Whether the condition of a secret holds under the values in force of the settings it reads. */
export function isSecretShown(
  plugin: PluginInfo,
  secret: SecretDeclaration,
  read: ReadPluginSetting,
  isRequested: (key: string) => boolean = () => false,
): boolean {
  if (!secret.when) return true
  const parent = plugin.settings.find((one) => one.key === secret.when?.setting)
  if (!parent || !isSettingShown(parent)) return false
  return secret.when.is.includes(toText(getValueInForce(parent, read, isRequested)))
}

function findSecretAnchor(
  secret: SecretDeclaration,
  shownKeys: ReadonlySet<string>,
): string | null {
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
