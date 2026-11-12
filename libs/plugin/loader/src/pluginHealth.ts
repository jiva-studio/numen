/**
 * The health items of one plugin, reported through the window under the id of that plugin.
 */
import type { HealthHost, HealthItem } from '@numen/plugin-sdk'

/** What the window does with the health items of the frontend of a plugin. */
export interface HealthReports {
  readonly reportHealth: (pluginId: string, item: HealthItem) => Promise<void>
  readonly clearHealth: (pluginId: string, key: string) => Promise<void>
}

/** The health host one plugin is given: every item is filed under its id. */
export const createPluginHealth = (reports: HealthReports, pluginId: string): HealthHost => ({
  report: (item) => reports.reportHealth(pluginId, item),
  clear: (key) => reports.clearHealth(pluginId, key),
})
