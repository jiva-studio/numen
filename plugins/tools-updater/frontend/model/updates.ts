/**
 * The checks for a newer build: when they run, what the last one found, and what the person did
 * with the announcement.
 *
 * A check that reaches no answer is not shown. It is repeated after a short wait, and then at the
 * interval the setting names.
 */
import { computed, ref, type ComputedRef, type Ref } from 'vue'
import type {
  ApiError,
  AppInfo,
  Failure,
  JsonValue,
  OpenExternalErrorCode,
  Result,
} from '@numen/plugin-sdk'
import type { UpdateApi } from '../api/wire'
import type { Status, Update } from '../api/updater'
import { RETRY_DELAYS_MS, SETTLE_DELAY_MS, getIntervalMs } from './waits'

/** Starts and stops the waits between checks. */
export interface Timer {
  set(run: () => void, delayMs: number): number
  clear(handle: number): void
}

export interface UpdateDeps {
  readonly api: UpdateApi
  readonly app: AppInfo
  readonly openExternal: (url: string) => Promise<Result<null, Failure<OpenExternalErrorCode>>>
  readonly getIntervalHours: () => JsonValue | undefined
  readonly timer: Timer
}

export interface Updates {
  readonly status: Readonly<Ref<Status>>
  readonly announced: ComputedRef<Update | null>
  readonly schedule: {
    /** Checks now and keeps checking at the interval of the setting. */
    start(): void
    /** Ends the waits between checks. */
    stop(): void
    /** Waits out the interval of the setting again, counting from now. */
    reschedule(): void
    /** Checks after a short wait, for a change of the feed. */
    checkSoon(): void
  }
  readonly announcement: {
    dismiss(): void
    openDownload(): Promise<Result<null, Failure<OpenExternalErrorCode>>>
    openNotes(): Promise<Result<null, Failure<OpenExternalErrorCode>>>
  }
}

export function createUpdates(deps: UpdateDeps): Updates {
  const status = ref<Status>({ state: 'unchecked' })
  const dismissedVersion = ref('')
  let handle: number | undefined
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

  const settle = (result: Result<Status, ApiError>) => {
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

  const open = (url: string | undefined): Promise<Result<null, Failure<OpenExternalErrorCode>>> =>
    url ? deps.openExternal(url) : Promise.resolve({ ok: true, value: null })

  return {
    status,
    announced,
    schedule: {
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
    },
    announcement: {
      dismiss: () => {
        dismissedVersion.value = status.value.update?.version ?? ''
      },
      openDownload: () => open(announced.value?.downloadUrl),
      openNotes: () => open(announced.value?.notesUrl),
    },
  }
}
