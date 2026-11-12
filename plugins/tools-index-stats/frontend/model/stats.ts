/**
 * What the title bar shows of the index: the last answer of the backend, kept fresh by asking every
 * two seconds while the window is in use.
 *
 * The core emits no event when the index changes, so the model asks. It asks only while the window is
 * visible and has the focus, and once more when the window returns to that state.
 */
import { ref, type Ref } from 'vue'
import type { ApiError, Result } from '@numen/plugin-sdk'
import type { StatsApi } from '../api/wire'
import type { Stats } from '../api/stats'

/** The wait between two asks. */
export const REFRESH_MS = 2_000

/** Starts and stops the waits between asks. */
export interface Timer {
  set(run: () => void, delayMs: number): number
  clear(handle: number): void
}

/** Whether the window is in use, and the call to make when it becomes so. */
export interface Activity {
  isActive(): boolean
  /** Calls the function each time the window becomes in use; returns the call that stops it. */
  onActive(run: () => void): () => void
}

/** Where the model is: before the first answer, with an answer, or with no index to show. */
export type Phase = 'loading' | 'ready' | 'unavailable'

export interface StatsDeps {
  readonly api: StatsApi
  readonly timer: Timer
  readonly activity: Activity
}

export interface IndexStats {
  readonly stats: Readonly<Ref<Stats | null>>
  readonly phase: Readonly<Ref<Phase>>
  /** Asks now and keeps asking every two seconds while the window is in use. */
  start(): void
  /** Ends the asking. */
  stop(): void
}

export function createIndexStats(deps: StatsDeps): IndexStats {
  const stats = ref<Stats | null>(null)
  const phase = ref<Phase>('loading')
  let handle: number | undefined
  let isStopped = true
  let isAsking = false
  let stopWatching: (() => void) | undefined

  const settle = (result: Result<Stats, ApiError>) => {
    isAsking = false
    if (isStopped) return
    if (result.ok) {
      stats.value = result.value
      phase.value = 'ready'
    } else {
      phase.value = 'unavailable'
    }
  }

  const ask = () => {
    if (isAsking || isStopped) return
    isAsking = true
    deps.api
      .get()
      .then(settle, () => settle({ ok: false, error: { code: 'unavailable', message: '' } }))
  }

  const wait = () => {
    if (isStopped) return
    handle = deps.timer.set(tick, REFRESH_MS)
  }

  function tick(): void {
    if (deps.activity.isActive()) ask()
    wait()
  }

  return {
    stats,
    phase,
    start: () => {
      isStopped = false
      stopWatching = deps.activity.onActive(ask)
      ask()
      wait()
    },
    stop: () => {
      isStopped = true
      if (handle !== undefined) deps.timer.clear(handle)
      handle = undefined
      stopWatching?.()
      stopWatching = undefined
    },
  }
}
