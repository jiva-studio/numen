import type { PluginInfo, PluginSettingsViewInfo } from '@/entities/settings'

/** The id of the tab that holds the plain settings rows and every view that names no tab. */
export const DEFAULT_TAB = 'settings'

/** One tab of the settings page of a plugin. */
export interface SettingsTabChoice {
  readonly id: string
  readonly text: string
}

/** The tab a view is drawn in. */
export function getViewTab(view: PluginSettingsViewInfo): string {
  return view.tab || DEFAULT_TAB
}

/** The tabs of the page: the default tab first, then the declared ones in their order. */
export function listSettingsTabs(
  plugin: Pick<PluginInfo, 'settingsTabs'>,
  defaultTitle: string,
): readonly SettingsTabChoice[] {
  return [
    { id: DEFAULT_TAB, text: defaultTitle },
    ...(plugin.settingsTabs ?? []).map((one) => ({ id: one.id, text: one.title })),
  ]
}
