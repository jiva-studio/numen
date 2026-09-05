/** A clock that only moves when a test says so. */
import type { Clock } from '../lib/clock'

export function stubClock() {
  let at = 0
  let next: ((now: number) => void) | null = null
  let handles = 0
  const cancelled: number[] = []

  const clock: Clock = {
    now: () => at,
    schedule: (run) => {
      next = run
      return ++handles
    },
    cancel: (handle) => {
      cancelled.push(handle)
      next = null
    },
  }

  return {
    clock,
    cancelled,
    get pending() {
      return next !== null
    },
    /** Advance to a moment and deliver the frame that was waiting for it. */
    tick(to: number) {
      at = to
      const run = next
      next = null
      run?.(at)
    },
    /** Every frame that is asked for, from a moment to the end of the movement. */
    run(from = 0) {
      for (let moment = from; this.pending; moment += 20) this.tick(moment)
    },
  }
}
