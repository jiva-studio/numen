/**
 * The health items of the plugins, as this window watches them, and the reports of the frontends of
 * the plugins.
 */
import { createHealthClient, type HealthReport, type PluginHealthItem } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'

export type { PluginHealthItem }

/** The health items of the plugins over `pluginHealth`. */
export const createHealthApi = (pluginHealth: WindowClients['pluginHealth']) => {
  const health = createHealthClient(pluginHealth)
  return {
    /** The items of every plugin until the signal aborts: at once and again on every change. */
    watchPluginHealth: (signal: AbortSignal): AsyncGenerator<readonly PluginHealthItem[]> =>
      health.watchHealth(signal),

    /** The reports of the frontends of the plugins, each filed under the id of its plugin. */
    healthReports: {
      reportHealth: (pluginId: string, item: HealthReport): Promise<void> =>
        health.reportHealth(pluginId, item),
      clearHealth: (pluginId: string, key: string): Promise<void> =>
        health.clearHealth(pluginId, key),
    },
  }
}

export type HealthApi = ReturnType<typeof createHealthApi>
