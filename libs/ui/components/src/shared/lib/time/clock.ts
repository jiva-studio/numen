/** The clock a component takes, with the schedule it runs work on. */
export interface Clock {
  readonly now: () => number
  readonly schedule: (run: (now: number) => void) => number
  readonly cancel: (handle: number) => void
}

/**
 * Schedules work to be drawn on screen. A component takes one as its default,
 * and a test supplies its own, so drawing happens when the test triggers it.
 */
export type Paint = (draw: () => void) => void
