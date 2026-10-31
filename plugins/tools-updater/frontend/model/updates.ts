/**
 * The checks for a newer build: when they run, what the last one found, and what the person did
 * with the announcement.
 *
 * A check that reaches no answer is not shown. It is repeated after a short wait, and then at the
 * interval the setting names.
 */
import { computed, ref, type ComputedRef, type Ref } from 'vue'
import type { AppInfo, Result } from '@numen/plugin-sdk'
import type { UpdateApi } from '../api/wire'
import type { Status, Update } from '../api/updater'

/** The hours between two checks where the setting holds no usable number. */
export const DEFAULT_INTERVAL_HOURS = 4

/** The waits before a check is repeated after one that reached no answer, the last repeating. */
export const RETRY_DELAYS_MS: readonly number[] = [10_000, 60_000, 15 * 60_000]

/** The wait after a setting that names the feed changed, which lets the backend apply it first. */
export const SETTLE_DELAY_MS = 1_000

const MS_PER_HOUR = 60 * 60 * 1000

/** Starts and stops the waits between checks. */
export interface Timer {
  set(run: () => void, delayMs: number): unknown
  clear(handle: unknown): void
}

export interface UpdateDeps {
  readonly api: UpdateApi
  readonly app: AppInfo
  readonly openExternal: (url: string) => Promise<Result<null>>
  /** The hours between two checks, as the setting holds them. */
  readonly getIntervalHours: () => unknown
  readonly timer: Timer
}

export interface Updates {
  readonly status: Readonly<Ref<Status>>
  /** The update to announce: the one found, unless the person dismissed it. */
  readonly announced: ComputedRef<Update | null>
  /** Checks now and keeps checking at the interval of the setting. */
  start(): void
  /** Ends the waits between checks. */
  stop(): void
  /** Waits out the interval of the setting again, counting from now. */
  reschedule(): void
  /** Checks after a short wait, for a change of the feed. */
  checkSoon(): void
  dismiss(): void
  openDownload(): Promise<Result<null>>
  openNotes(): Promise<Result<null>>
}

/** The wait between two checks the setting names; the default where it holds no usable number. */
export function getIntervalMs(hours: unknown): number {
  const count = typeof hours === 'number' ? hours : Number(hours)
  return (Number.isFinite(count) && count >= 1 ? count : DEFAULT_INTERVAL_HOURS) * MS_PER_HOUR
}

export function createUpdates(deps: UpdateDeps): Updates {
  const status = ref<Status>({ state: 'unchecked' })
  const dismissedVersion = ref('')
  let handle: unknown
  let isStopped = true
  let failureCount = 0

  const announced = computed(() => {
    const update = status.value.update
    return status.value.state === 'available' && update && update.version !== dismissedVersion.value
      ? update
      : null
  })

  const clearWait = () => {
    if (handle !== undefined) deps.timer.clear(handle)
    handle = undefined
  }

  const wait = (delayMs: number) => {
    clearWait()
    if (!isStopped) handle = deps.timer.set(runCheck, delayMs)
  }

  const getRetryDelay = (): number => {
    const index = Math.min(failureCount, RETRY_DELAYS_MS.length) - 1
    return Math.min(RETRY_DELAYS_MS[index] ?? 0, getIntervalMs(deps.getIntervalHours()))
  }

  const retry = () => {
    failureCount += 1
    wait(getRetryDelay())
  }

  const settle = (result: Result<Status, unknown>) => {
    if (isStopped) return
    if (!result.ok || result.value.state === 'unreachable') {
      retry()
      return
    }
    status.value = result.value
    failureCount = 0
    wait(getIntervalMs(deps.getIntervalHours()))
  }

  function runCheck(): void {
    deps.api.check(deps.app).then(settle, retry)
  }

  const open = (url: string | undefined): Promise<Result<null>> =>
    url ? deps.openExternal(url) : Promise.resolve({ ok: true, value: null })

  return {
    status,
    announced,
    start: () => {
      isStopped = false
      failureCount = 0
      runCheck()
    },
    stop: () => {
      isStopped = true
      clearWait()
    },
    reschedule: () => wait(getIntervalMs(deps.getIntervalHours())),
    checkSoon: () => {
      failureCount = 0
      status.value = { state: 'unchecked' }
      wait(SETTLE_DELAY_MS)
    },
    dismiss: () => {
      dismissedVersion.value = status.value.update?.version ?? ''
    },
    openDownload: () => open(announced.value?.downloadUrl),
    openNotes: () => open(announced.value?.notesUrl),
  }
}
