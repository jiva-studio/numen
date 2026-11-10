import type { PluginInfo } from '@/entities/settings'
import { isSecretShown } from './enums'
import type { JsonValue } from '@/shared/api/json'

/** Whether the plugin declares anything its page draws: settings, views of its own, or a secret that is shown. */
export function hasPluginSettings(
  plugin: PluginInfo,
  read: (key: string) => JsonValue | undefined,
): boolean {
  return (
    plugin.settings.length > 0 ||
    (plugin.settingsViews ?? []).length > 0 ||
    plugin.secrets.some((one) => isSecretShown(plugin, one, read))
  )
}
