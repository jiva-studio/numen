import { isShown, type PluginInfo } from '@/entities/settings'

/** Whether the plugin declares anything its page draws: settings, views of its own, or a secret that is shown. */
export function hasPluginSettings(plugin: PluginInfo, read: (key: string) => unknown): boolean {
  return (
    plugin.settings.length > 0 ||
    (plugin.settingsViews ?? []).length > 0 ||
    plugin.secrets.some((one) => isShown(plugin, one.when, read))
  )
}
