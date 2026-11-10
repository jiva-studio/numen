/** The settings tab: everything in numen.json a user can change. */
export { default as SettingsTab } from './ui/SettingsTab.vue'
export { useSettingsTab } from './kind'
export { useSettings } from './model/useSettings'
export type {
  Installation,
  InstallationPlugin,
  InstallationSecrets,
  InstallationSettings,
} from './types'
