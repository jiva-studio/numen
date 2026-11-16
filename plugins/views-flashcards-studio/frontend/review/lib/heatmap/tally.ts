/** What one day behind came to. */
export interface Tally {
  readonly reviewed: number
  readonly again: number
  readonly hard: number
  readonly good: number
  readonly easy: number
  readonly tested: number
  readonly recalled: number
}

/** A day with no answers, which is the value of an empty cell. */
export const NOTHING: Tally = {
  reviewed: 0,
  again: 0,
  hard: 0,
  good: 0,
  easy: 0,
  tested: 0,
  recalled: 0,
}
