import type { IdentityView, ViewComponent } from '../types'
import type { PluginInfo, PluginSettingsViewInfo } from '@/entities/settings'
import { getValueInForce, type ReadPluginSetting } from './enums'
import { DEFAULT_TAB, getViewTab } from './settingsTabs'
import type { JsonValue } from '@/shared/api/json'

/** The page of a plugin view that the page "Identity and licenses" draws instead of the plugin's own page. */
export const IDENTITY_PAGE = 'identity'

/** A view the plugin declares and its frontend registered. */
export interface MountedSettingsView extends ViewComponent {
  readonly id: string
}

/** The declared views of the tab of the settings form that the frontend registered, in declared order. */
export function getRegisteredViews(
  declared: readonly PluginSettingsViewInfo[],
  registered: ReadonlyMap<string, ViewComponent>,
  tab = DEFAULT_TAB,
): readonly MountedSettingsView[] {
  return listViews(
    declared.filter((one) => one.page !== IDENTITY_PAGE && getViewTab(one) === tab),
    registered,
  )
}

/** The views of the enabled plugins that belong to the page "Identity and licenses", in the order of the plugins. */
export function getIdentityViews(
  plugins: readonly PluginInfo[],
  getRegistered: (pluginId: string) => ReadonlyMap<string, ViewComponent>,
): readonly IdentityView[] {
  return plugins
    .filter((plugin) => plugin.isActive)
    .flatMap((plugin) =>
      listViews(
        (plugin.settingsViews ?? []).filter((one) => one.page === IDENTITY_PAGE),
        getRegistered(plugin.id),
      ).map((view) => ({ id: view.id, component: view.component, pluginId: plugin.id })),
    )
}

function listViews(
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
