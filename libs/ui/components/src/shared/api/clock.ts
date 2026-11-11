/** The machine's clock and frame schedule, and the only place in the library that reads them. */
import type { Clock } from '../lib/time/clock'

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
 * The next frame where there is one, and now where there is none, for work
 * that has nothing to cancel. A component takes this as a default and a test
 * hands in its own.
 */
export const onNextFrame = (run: () => void): void => {
  if (typeof globalThis.requestAnimationFrame === 'function') browserClock.schedule(run)
  else run()
}
