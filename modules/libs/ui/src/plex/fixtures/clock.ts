/** A clock that only moves when a test says so. */
import type { Environment } from '../transition'

export function stubEnvironment() {
  let clock = 0
  let next: ((now: number) => void) | null = null
  let handles = 0
  const cancelled: number[] = []

  const environment: Environment = {
    now: () => clock,
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
    environment,
    cancelled,
    get pending() {
      return next !== null
    },
    /** Advance to a moment and deliver the frame that was waiting for it. */
    tick(to: number) {
      clock = to
      const run = next
      next = null
      run?.(clock)
    },
    /** Every frame that is asked for, from a moment to the end of the movement. */
    run(from = 0) {
      for (let at = from; this.pending; at += 20) this.tick(at)
    },
  }
}
