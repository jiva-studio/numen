import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'

/** Whether the settings page draws a row for the setting: connections and slots have pages of their own. */
function isRowSetting(setting: PluginSettingInfo): boolean {
  return setting.type !== 'connections' && setting.type !== 'slots'
}

/** Whether any of the settings is drawn as a row. */
export function hasRowSettings(settings: readonly PluginSettingInfo[]): boolean {
  return settings.some(isRowSetting)
}

/** Whether the plugin declares anything its page draws: settings or views of its own. */
export function hasPluginSettings(plugin: PluginInfo): boolean {
  return hasRowSettings(plugin.settings) || (plugin.settingsViews ?? []).length > 0
}
