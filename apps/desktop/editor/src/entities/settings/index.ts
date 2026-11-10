/** What the settings file holds, and how the window is drawn from it. */
export { DESIGNED, MODES, SCHEMES } from './lib/appearance'
export type {
  Model,
  SettingEdit,
  PluginInfo,
  PluginSettingInfo,
  PluginSettingsViewInfo,
  PluginSettingOption,
  SecretDeclaration,
  Secret,
  SecretMenu,
  SecretSection,
} from './lib/configuration'
export { SETTINGS_WORDS } from './words'
export { createSettingsStore } from './model/store'
export type { SettingsStore } from './model/store'
export { themes } from './api/theme'
export type { Appearance, Mode, Scales, Sizes, Themes } from './api/theme'
export { createSettingFields } from './lib/fields'
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
export { createSecretStore } from './model/secrets'
export { createPluginRetry } from './model/retry'
export { getFirstSecretName, getNextSecretName } from './model/serviceNames'
export { getServiceInfo } from './api/services'
export { createChangeFollower } from './model/changeFollower'
export { choosePluginPath, getPluginData, deletePluginData } from './api/settings'
