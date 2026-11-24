/** Runs a task again after a delay until it is stopped; the next run is set by the task itself. */
export function createPoller(delayMs: number, onFail: (failure: unknown) => void) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let isStopped = false
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
  return { stop, runAgain }
}
