import type { PluginInfo } from '@/entities/settings'

/** Whether the plugin declares anything its page draws: settings or views of its own. */
export function hasPluginSettings(plugin: PluginInfo): boolean {
  return plugin.settings.some((one) => one.type !== 'connections') || (plugin.settingsViews ?? []).length > 0
}
