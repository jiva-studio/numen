/** A clock that advances only when a test advances it. */
import type { Clock } from '@/shared/lib/time/clock'

export function stubClock() {
  let current = 0
  let next: ((now: number) => void) | null = null
  let handles = 0
  const cancelled: number[] = []

  const clock: Clock = {
    now: () => current,
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
      current = to
      const run = next
      next = null
      run?.(current)
    },
    /** Every frame that is requested, from a moment to the end of the movement. */
    run(from = 0) {
      for (let moment = from; this.pending; moment += 20) this.tick(moment)
    },
  }
}
