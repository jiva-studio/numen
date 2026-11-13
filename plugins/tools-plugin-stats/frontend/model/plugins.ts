/**
 * What the title bar shows of the plugins that run: the last list the core pushed, in the order a
 * person reads it, and the time now on the clock of the core. The core sends a new list on every
 * change, so the model only listens; between two lists it moves the time on by a monotonic clock,
 * once a second, so the times in the list count.
 */
import { ref, shallowRef, type Ref } from 'vue'
import type { Disposable, ProcessHost } from '@numen/plugin-sdk'
import type { PluginProcess } from '../types'

/** How often the time moves on between two lists. */
export const TICK_MS = 1_000

/** Where the model is: before the first list, with a list, or with no list to show. */
export type Phase = 'loading' | 'ready' | 'unavailable'

/** A clock that only moves forward, in milliseconds from any start. */
export interface MonotonicClock {
  now(): number
}

/** Runs a function at a fixed interval until stopped. */
export interface Ticker {
  start(run: () => void, intervalMs: number): number
  stop(handle: number): void
}

/** What the model is made of; the host is absent where the window gives no list. */
export interface RunningPluginsDeps {
  readonly host: ProcessHost | undefined
  readonly clock: MonotonicClock
  readonly ticker: Ticker
}

/**
 * The list, the phase, and the time now on the clock of the core in milliseconds since the Unix
 * epoch.
 */
export interface RunningPlugins {
  readonly processes: Readonly<Ref<readonly PluginProcess[]>>
  readonly phase: Readonly<Ref<Phase>>
  readonly now: Readonly<Ref<number>>
  /** Listens to the core and moves the time on. */
  start(): void
  /** Ends the listening and the moving on. */
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

export function createRunningPlugins(deps: RunningPluginsDeps): RunningPlugins {
  const { host, clock, ticker } = deps
  const processes = shallowRef<readonly PluginProcess[]>([])
  const phase = ref<Phase>('loading')
  const now = ref(0)
  // The time of the last list, and the monotonic time it arrived at. A list sent again unchanged
  // carries the time of the first, so it keeps the anchor.
  let anchor = { time: 0, at: 0 }
  let watching: Disposable | undefined
  let tick: number | undefined

  const moveOn = () => {
    now.value = anchor.time + (clock.now() - anchor.at)
  }

  const take = (list: readonly PluginProcess[], time: number) => {
    if (time !== anchor.time) anchor = { time, at: clock.now() }
    processes.value = sortProcesses(list)
    phase.value = 'ready'
    moveOn()
  }

  return {
    processes,
    phase,
    now,
    start: () => {
      if (!host) {
        phase.value = 'unavailable'
        return
      }
      try {
        watching = host.watch(take)
      } catch {
        phase.value = 'unavailable'
        return
      }
      tick = ticker.start(moveOn, TICK_MS)
    },
    stop: () => {
      watching?.dispose()
      watching = undefined
      if (tick !== undefined) ticker.stop(tick)
      tick = undefined
    },
  }
}
