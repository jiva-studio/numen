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
