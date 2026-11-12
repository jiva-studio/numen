/**
 * What the plugins say about whether they work, in the terms of the windows: the health items of
 * every plugin, and the report and clear a window makes for the frontend of a plugin.
 */
import type { Client } from '@connectrpc/connect'
import { HealthSeverity, type HealthItem, type PluginHealthService } from '@numen/protocol'

/** How bad a health item is. */
export type HealthSeverityName = 'warning' | 'error'

/** The command a person runs to fix what an item reports. An empty title reads the command's own. */
export interface HealthAction {
  readonly command: string
  readonly values: Readonly<Record<string, string>>
  readonly title: string
}

/** One concern of one plugin that does not work. */
export interface PluginHealthItem {
  readonly pluginId: string
  readonly key: string
  readonly severity: HealthSeverityName
  readonly text: string
  readonly detail: string
  readonly action?: HealthAction | undefined
}

/** An item a frontend reports; the window names the plugin. */
export interface HealthReport {
  readonly key: string
  readonly severity: HealthSeverityName
  readonly text: string
  readonly detail?: string | undefined
  readonly action?:
    | {
        readonly command: string
        readonly values?: Readonly<Record<string, string>> | undefined
        readonly title?: string | undefined
      }
    | undefined
}

/** The calls of PluginHealthService a window makes. */
export type HealthRpc = Pick<
  Client<typeof PluginHealthService>,
  'watchHealth' | 'reportHealth' | 'clearHealth'
>

/** The health items of the plugins, as a window reads and reports them. */
export interface HealthClient {
  /** The items of every plugin until the signal aborts: at once and again on every change. */
  watchHealth(signal: AbortSignal): AsyncGenerator<readonly PluginHealthItem[]>
  /** Reports an item of the frontend of the plugin. Rejects an item the core refuses. */
  reportHealth(pluginId: string, item: HealthReport): Promise<void>
  /** Clears the item of the plugin under the key. */
  clearHealth(pluginId: string, key: string): Promise<void>
}

const SEVERITY: Readonly<Record<HealthSeverityName, HealthSeverity>> = {
  warning: HealthSeverity.WARNING,
  error: HealthSeverity.ERROR,
}

const readItem = (item: HealthItem): PluginHealthItem => ({
  pluginId: item.pluginId,
  key: item.key,
  severity: item.severity === HealthSeverity.WARNING ? 'warning' : 'error',
  text: item.text,
  detail: item.detail,
  action: item.action
    ? { command: item.action.command, values: { ...item.action.values }, title: item.action.title }
    : undefined,
})

/** The health items over the plugin health service. */
export function createHealthClient(rpc: HealthRpc): HealthClient {
  return {
    watchHealth: async function* (signal) {
      for await (const update of rpc.watchHealth({}, { signal })) {
        yield update.items
          .filter((item) => item.severity !== HealthSeverity.OK)
          .map(readItem)
      }
    },
    reportHealth: async (pluginId, item) => {
      await rpc.reportHealth({
        item: {
          pluginId,
          key: item.key,
          severity: SEVERITY[item.severity],
          text: item.text,
          detail: item.detail ?? '',
          action: item.action
            ? {
                command: item.action.command,
                values: { ...item.action.values },
                title: item.action.title ?? '',
              }
            : undefined,
        },
      })
    },
    clearHealth: async (pluginId, key) => {
      await rpc.clearHealth({ pluginId, key })
    },
  }
}

