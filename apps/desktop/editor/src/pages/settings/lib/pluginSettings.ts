import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'
import { IDENTITY_PAGE } from './settingsViews'

function isRowSetting(setting: PluginSettingInfo): boolean {
  return setting.type !== 'connections' && setting.type !== 'slots'
}

/** Whether any of the settings is drawn as a row. */
export function hasRowSettings(settings: readonly PluginSettingInfo[]): boolean {
  return settings.some(isRowSetting)
}

/** Whether the plugin declares anything its page draws: settings it shows or views of its own. */
export function hasPluginSettings(plugin: PluginInfo): boolean {
  return (
    hasRowSettings(plugin.settings.filter((one) => one.isShown !== false)) ||
    (plugin.settingsViews ?? []).some((one) => one.page !== IDENTITY_PAGE)
  )
}
