/** What the settings file holds, and how the window is drawn from it. */
export {
  DESIGNED,
  MODES,
  SCHEMES,
} from './lib/appearance'
export type {
  Configuration,
  Model,
  Presence,
  SettingEdit,
  PluginInfo,
  PluginSettingInfo,
  PluginSettingsViewInfo,
  PluginSettingOption,
  PluginSettingCondition,
  PluginSecret,
} from './lib/configuration'
export { getSettingAt, createSettingsStore } from './model/store'
export { themes } from './api/theme'
export type { Appearance, Bounds, Mode, Ranges, Scales, Sizes, Theme, Themes } from './api/theme'
export { createSettingFields } from './lib/fields'
export type { SettingFields, SettingSource } from './lib/fields'
export { write } from './lib/write'
export { createSecretStore } from './model/secrets'
export { createPluginRetry } from './model/retry'
export type { RetryDeps, RetryOptions, RetryWords } from './model/retry'
export type { SecretDeps, SecretOptions, SecretWords } from './model/secrets'
