/** What the settings file holds, and how the window is drawn from it. */
export { DESIGNED, MODES, SCHEMES } from './lib/appearance'
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
  SecretDeclaration,
  Secret,
  SecretMenu,
  DeclarationKey,
  SecretSection,
} from './lib/configuration'
export { getSettingAt, createSettingsStore } from './model/store'
export { themes } from './api/theme'
export type { Appearance, Bounds, Mode, Ranges, Scales, Sizes, Theme, Themes } from './api/theme'
export { createSettingFields } from './lib/fields'
export type { SettingFields, SettingSource } from './lib/fields'
export { write } from './lib/write'
export {
  MAX_NAME_LENGTH,
  getMoveTarget,
  getNewGroupName,
  isAllowed,
  isGroupNameTaken,
  isNameTooLong,
  trimName,
} from './lib/secretNames'
export type { MoveTarget, PermissionKey } from './lib/secretNames'
export { createSecretStore } from './model/secrets'
export { createPluginRetry } from './model/retry'
export type { RetryOptions, RetryWords } from './model/retry'
export type { SecretOptions, SecretWords } from './model/secrets'
export { getFirstSecretName, getNextSecretName } from './model/serviceNames'
export { getServiceInfo } from './api/services'
export { watchSettings, choosePluginPath, getPluginData, deletePluginData } from './api/settings'
