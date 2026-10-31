import {
  getValue,
  type PluginInfo,
  type PluginSettingsViewInfo,
  type ReadPluginSetting,
} from '@/entities/settings'

/** A view the plugin declares and its frontend registered. */
export interface MountedSettingsView {
  readonly id: string
  readonly component: unknown
}

/** The declared views that the frontend registered, in declared order. */
export function getRegisteredViews(
  declared: readonly PluginSettingsViewInfo[],
  registered: ReadonlyMap<string, { readonly component: unknown }>,
): readonly MountedSettingsView[] {
  const views: MountedSettingsView[] = []
  for (const one of declared) {
    const entry = registered.get(one.id)
    if (entry) views.push({ id: one.id, component: entry.component })
  }
  return views
}

/** The value of every declared setting of the plugin, by key. */
export function getSettingValues(
  plugin: PluginInfo,
  read: ReadPluginSetting,
): Readonly<Record<string, unknown>> {
  const values: Record<string, unknown> = {}
  for (const one of plugin.settings) values[one.key] = getValue(plugin, one, read).value
  return values
}
