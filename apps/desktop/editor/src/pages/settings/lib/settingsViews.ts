import type { ViewComponent } from '../types'
import type { PluginInfo, PluginSettingsViewInfo } from '@/entities/settings'
import { getValueInForce, type ReadPluginSetting } from './enums'
import type { JsonValue } from '@/shared/api/json'

/** A view the plugin declares and its frontend registered. */
export interface MountedSettingsView extends ViewComponent {
  readonly id: string
}

/** The declared views that the frontend registered, in declared order. */
export function getRegisteredViews(
  declared: readonly PluginSettingsViewInfo[],
  registered: ReadonlyMap<string, ViewComponent>,
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
  isRequested: (key: string) => boolean = () => false,
): Readonly<Record<string, JsonValue | undefined>> {
  const values: Record<string, JsonValue | undefined> = {}
  for (const one of plugin.settings) values[one.key] = getValueInForce(one, read, isRequested)
  return values
}
