/** The clock, and the only place in the library that reads the current time. */

export interface Clock {
  readonly now: () => number
  readonly schedule: (run: (now: number) => void) => number
  readonly cancel: (handle: number) => void
}

/** Milliseconds on the monotonic clock. The only function that reads it; tests control it with fake timers. */
export function getMonotonicNow(): number {
  return performance.now()
}

export const browserClock: Clock = {
  now: getMonotonicNow,
  schedule: (run) => globalThis.requestAnimationFrame(run),
  cancel: (handle) => globalThis.cancelAnimationFrame(handle),
}

/**
 * Schedules work to be drawn on screen. A component takes one as its default,
 * and a test supplies its own, so drawing happens when the test triggers it.
 */
export type Paint = (draw: () => void) => void

/**
 * The next frame where there is one, and now where there is none, for work
 * that has nothing to cancel. A component takes this as a default and a test
 * hands in its own.
 */
export const onNextFrame = (run: () => void): void => {
  if (typeof globalThis.requestAnimationFrame === 'function') browserClock.schedule(run)
  else run()
}
