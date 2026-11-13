/**
 * What the title bar shows of the plugins that run: the last list the core pushed, in the order a
 * person reads it. The core sends a new list on every change, so the model only listens.
 */
import { ref, shallowRef, type Ref } from 'vue'
import type { Disposable, ProcessHost } from '@numen/plugin-sdk'
import type { PluginProcess } from '../types'

/** Where the model is: before the first list, with a list, or with no list to show. */
export type Phase = 'loading' | 'ready' | 'unavailable'

export interface RunningPlugins {
  readonly processes: Readonly<Ref<readonly PluginProcess[]>>
  readonly phase: Readonly<Ref<Phase>>
  /** Listens to the core. */
  start(): void
  /** Ends the listening. */
  stop(): void
}

const isHeld = (one: PluginProcess) => one.leaseCount > 0 || one.idleStopTime === undefined

const compareNames = (a: PluginProcess, b: PluginProcess) =>
  (a.pluginName || a.pluginId).localeCompare(b.pluginName || b.pluginId) ||
  a.pluginId.localeCompare(b.pluginId)

/**
 * The plugins in the order a person reads them: the ones in use first, by name, then the idle ones,
 * the one the core stops first at the top.
 */
export function sortProcesses(processes: readonly PluginProcess[]): PluginProcess[] {
  return [...processes].sort((a, b) => {
    if (isHeld(a) !== isHeld(b)) return isHeld(a) ? -1 : 1
    if (isHeld(a)) return compareNames(a, b)
    return (a.idleStopTime ?? 0) - (b.idleStopTime ?? 0) || compareNames(a, b)
  })
}

/** The running plugins over the host of the window, which is absent where the window gives none. */
export function createRunningPlugins(host: ProcessHost | undefined): RunningPlugins {
  const processes = shallowRef<readonly PluginProcess[]>([])
  const phase = ref<Phase>('loading')
  let watching: Disposable | undefined

  return {
    processes,
    phase,
    start: () => {
      if (!host) {
        phase.value = 'unavailable'
        return
      }
      try {
        watching = host.watch((list) => {
          processes.value = sortProcesses(list)
          phase.value = 'ready'
        })
      } catch {
        phase.value = 'unavailable'
      }
    },
    stop: () => {
      watching?.dispose()
      watching = undefined
    },
  }
}
