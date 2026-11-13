/**
 * The plugins that run, in the terms of the windows.
 */
import type { Client } from '@connectrpc/connect'
import {
  HealthSeverity,
  type PluginProcess as PluginProcessMessage,
  type PluginService,
} from '@numen/protocol'

/**
 * A plugin that runs. Times are milliseconds since the Unix epoch; `idleStopTime` is absent while
 * the plugin holds a lease.
 */
export interface PluginProcess {
  readonly pluginId: string
  readonly pluginName: string
  readonly startTime: number
  readonly lastUseTime: number
  readonly leaseCount: number
  readonly isInProcess: boolean
  readonly idleStopTime?: number
  readonly health: 'ok' | 'warning' | 'error'
  readonly healthText: string
}

/** The calls of PluginService a window makes about the plugins that run and their processes. */
export type ProcessRpc = Pick<
  Client<typeof PluginService>,
  'watchPluginProcesses' | 'releaseProcess'
>

/** The plugins that run, with the time of the core when it took the list. */
export interface PluginProcessList {
  readonly time: number
  readonly processes: readonly PluginProcess[]
}

/** The plugins that run, as a window watches them. */
export interface ProcessClient {
  /** The plugins that run until the signal aborts: at once and again on every change. */
  watchProcesses(signal: AbortSignal): AsyncGenerator<PluginProcessList>
  /**
   * Tells the core that the backend of the plugin has finished its work. The core stops the process
   * at once where no call into it is open, and else as soon as the last call ends.
   */
  releaseProcess(pluginId: string): Promise<void>
}

const HEALTH: Readonly<Partial<Record<HealthSeverity, PluginProcess['health']>>> = {
  [HealthSeverity.WARNING]: 'warning',
  [HealthSeverity.ERROR]: 'error',
}

const readProcess = (one: PluginProcessMessage): PluginProcess => ({
  pluginId: one.pluginId,
  pluginName: one.pluginName,
  startTime: Number(one.startTimeUnixMs),
  lastUseTime: Number(one.lastUseTimeUnixMs),
  leaseCount: one.leaseCount,
  isInProcess: one.isInProcess,
  ...(one.idleStopTimeUnixMs > 0n && { idleStopTime: Number(one.idleStopTimeUnixMs) }),
  health: HEALTH[one.health] ?? 'ok',
  healthText: one.healthText,
})

/** The plugins that run over the plugin service. */
export function createProcessClient(rpc: ProcessRpc): ProcessClient {
  return {
    watchProcesses: async function* (signal) {
      for await (const update of rpc.watchPluginProcesses({}, { signal })) {
        yield { time: Number(update.timeUnixMs), processes: update.processes.map(readProcess) }
      }
    },
    releaseProcess: async (pluginId) => {
      await rpc.releaseProcess({ pluginId })
    },
  }
}
