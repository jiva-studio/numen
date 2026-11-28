import type { PluginSettingsViewInfo } from '@/entities/settings'

/** The id of the tab that holds the plain settings rows and every view that names no tab. */
export const DEFAULT_TAB = 'settings'

/** The tab a view is drawn in. */
export function getViewTab(view: PluginSettingsViewInfo): string {
  return view.tab || DEFAULT_TAB
}
