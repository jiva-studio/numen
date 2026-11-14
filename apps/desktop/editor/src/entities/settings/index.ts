/** What the settings file holds, and how the window is drawn from it. */
export type {
  SettingEdit,
  PluginInfo,
  PluginSettingInfo,
  PluginSettingsViewInfo,
  PluginSettingOption,
} from './lib/configuration'
export { SETTINGS_WORDS } from './words'
export { createSettingsStore } from './model/store'
export type { SettingsStore } from './model/store'
export { createThemes, type ThemeClient } from './api/theme'
export { createSettingFields } from './lib/fields'
export { write } from './lib/write'
export { createPluginRetry } from './model/retry'
export { createChangeFollower } from './model/changeFollower'
export { createSettingsApi, type SettingsApi } from './api/settings'
export type { SettingsStoreApi } from './model/store'
