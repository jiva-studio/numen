/** The clock, and the only thing in the library that knows what time it is. */

export interface Environment {
  /** Milliseconds, monotonic. Only differences are used. */
  readonly now: () => number
  readonly schedule: (run: (now: number) => void) => number
  readonly cancel: (handle: number) => void
}

export const browserEnvironment: Environment = {
  now: () => performance.now(),
  schedule: (run) => requestAnimationFrame(run),
  cancel: (handle) => cancelAnimationFrame(handle),
}

/**
 * The next frame where there is one, and now where there is none, for work
 * that has nothing to cancel. A component takes this as a default and a test
 * hands in its own.
 */
export const onNextFrame = (run: () => void): void => {
  if (typeof requestAnimationFrame === 'function') browserEnvironment.schedule(run)
  else run()
}
