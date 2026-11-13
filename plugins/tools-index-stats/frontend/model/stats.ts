/**
 * What the title bar shows of the index: the last answer of the backend.
 *
 * The core emits no event with the counts of the index, so the model asks the backend. It asks once at
 * start and again each time a background task of the window starts or ends, which is when the counts
 * move. Only while the popover is open, the window is visible and has the focus, it asks every two
 * seconds, which is what the backend measures the speed of embedding from. With the popover closed no
 * timer runs, so the backend is called only on those changes.
 */
import { ref, type Ref } from 'vue'
import type {
  ApiError,
  BackgroundTask,
  BackgroundTaskHost,
  Disposable,
  Result,
} from '@numen/plugin-sdk'
import type { StatsApi } from '../api/wire'
import type { Stats } from '../api/stats'

/** The wait between two asks while the popover is open. */
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
  readonly tasks?: Pick<BackgroundTaskHost, 'watch'> | undefined
}

export interface IndexStats {
  readonly stats: Readonly<Ref<Stats | null>>
  readonly phase: Readonly<Ref<Phase>>
  /** Asks now, and again each time a background task starts or ends. */
  start(): void
  /** Asks now and every two seconds while the window is in use, until close. */
  open(): void
  /** Ends the asks every two seconds. */
  close(): void
  /** Ends every ask. */
  stop(): void
}

function getRunningKey(tasks: readonly BackgroundTask[]): string {
  return tasks
    .filter((one) => one.state === 'running' || one.state === 'cancelling')
    .map((one) => one.id)
    .sort()
    .join('\n')
}

export function createIndexStats(deps: StatsDeps): IndexStats {
  const stats = ref<Stats | null>(null)
  const phase = ref<Phase>('loading')
  let handle: number | undefined
  let isStopped = true
  let isOpen = false
  let isAsking = false
  let runningKey: string | undefined
  let stopWatching: (() => void) | undefined
  let taskWatch: Disposable | undefined

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

  const clearWait = () => {
    if (handle !== undefined) deps.timer.clear(handle)
    handle = undefined
  }

  const wait = () => {
    if (isStopped || !isOpen) return
    handle = deps.timer.set(tick, REFRESH_MS)
  }

  function tick(): void {
    handle = undefined
    if (deps.activity.isActive()) ask()
    wait()
  }

  const onTasks = (tasks: readonly BackgroundTask[]) => {
    const key = getRunningKey(tasks)
    const isChange = runningKey !== undefined && key !== runningKey
    runningKey = key
    if (isChange) ask()
  }

  const close = () => {
    isOpen = false
    clearWait()
    stopWatching?.()
    stopWatching = undefined
  }

  return {
    stats,
    phase,
    start: () => {
      isStopped = false
      taskWatch = deps.tasks?.watch(onTasks)
      ask()
    },
    open: () => {
      if (isStopped || isOpen) return
      isOpen = true
      stopWatching = deps.activity.onActive(ask)
      ask()
      wait()
    },
    close,
    stop: () => {
      close()
      isStopped = true
      taskWatch?.dispose()
      taskWatch = undefined
      runningKey = undefined
    },
  }
}
