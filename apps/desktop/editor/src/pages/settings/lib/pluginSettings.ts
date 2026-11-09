import type { PluginInfo } from '@/entities/settings'

/** Whether the plugin declares anything its page draws: settings or views of its own. */
export function hasPluginSettings(plugin: PluginInfo): boolean {
  return plugin.settings.length > 0 || (plugin.settingsViews ?? []).length > 0
}
