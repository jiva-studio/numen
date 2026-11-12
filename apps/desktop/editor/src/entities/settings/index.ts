/** What the settings file holds, and how the window is drawn from it. */
export type {
  Model,
  SettingEdit,
  PluginInfo,
  PluginSettingInfo,
  PluginSettingsViewInfo,
  PluginSettingOption,
} from './lib/configuration'
export { SETTINGS_WORDS } from './words'
export { createSettingsStore } from './model/store'
export type { SettingsStore } from './model/store'
export { themes } from './api/theme'
export { createSettingFields } from './lib/fields'
export { write } from './lib/write'
export { createPluginRetry } from './model/retry'
export { createChangeFollower } from './model/changeFollower'
export { choosePluginPath, getPluginData, deletePluginData } from './api/settings'
