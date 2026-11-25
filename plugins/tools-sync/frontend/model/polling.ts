/** How many runs in a row may fail before the poller gives up. */
export const MAX_FAILURES = 5

/**
 * Runs a task again after a delay until it is stopped; the next run is set by the task itself. A task that
 * reports its failures with `countFailure` is not run again once `MAX_FAILURES` of them came in a row.
 */
export function createPoller(delayMs: number, onFail: (failure: unknown) => void) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let isStopped = false
  let failures = 0
  const stop = (): void => {
    isStopped = true
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }
  const runAgain = (task: () => Promise<void>): void => {
    if (isStopped) return
    if (timer !== undefined) clearTimeout(timer)
    timer = setTimeout(() => {
      task().catch(onFail)
    }, delayMs)
  }
  const countFailure = (): boolean => {
    failures += 1
    if (failures < MAX_FAILURES) return false
    stop()
    return true
  }
  const countSuccess = (): void => {
    failures = 0
  }
  return { stop, runAgain, countFailure, countSuccess }
}
