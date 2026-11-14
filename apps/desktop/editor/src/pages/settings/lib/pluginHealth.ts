/**
 * The health items of the plugins as the plugin manager draws them, and the settings their fixes
 * name.
 */
import type { PluginHealth } from '@numen/plugin-manager'
import type { PluginHealthItem } from '@/entities/plugin'
import { WORDS as words } from '../words'

/** The items of one plugin as the manager draws them; a fix the window cannot run has no button. */
export function getManagedHealth(
  items: readonly PluginHealthItem[],
  pluginId: string,
  canRunCommand: (command: string) => boolean,
): readonly PluginHealth[] {
  return items
    .filter((item) => item.pluginId === pluginId)
    .map((item) => ({
      key: item.key,
      severity: item.severity,
      text: item.text,
      detail: item.detail,
      actionLabel:
        item.action && canRunCommand(item.action.command)
          ? item.action.title || words.healthAction
          : '',
    }))
}

/** The items of every plugin as the manager draws them, by the id of the plugin. */
export function groupManagedHealth(
  items: readonly PluginHealthItem[],
  canRunCommand: (command: string) => boolean,
): Readonly<Record<string, readonly PluginHealth[]>> {
  const ids = [...new Set(items.map((item) => item.pluginId))]
  return Object.fromEntries(ids.map((id) => [id, getManagedHealth(items, id, canRunCommand)]))
}

/** The keys of the settings the fixes of the items of the plugin name. */
export function getMarkedSettings(
  items: readonly PluginHealthItem[],
  pluginId: string,
): readonly string[] {
  return items
    .filter((item) => item.pluginId === pluginId)
    .map((item) => item.action?.values['setting'] ?? '')
    .filter((setting) => setting !== '')
}
