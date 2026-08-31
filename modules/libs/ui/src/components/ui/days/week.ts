/**
 * The seven days, as they are drawn and as they are named.
 *
 * Pure: which day the week is turned to start on is arithmetic over a list,
 * and the names are the caller's to hand in.
 */

/** One day of the week. */
export interface Day {
  /** The caller's own identifier, handed back as given. */
  readonly id: string
  /** What is drawn on the chip: a letter or two. */
  readonly short: string
  /** What the day is called, which is what a screen reader says. */
  readonly long: string
}

/** The week as English names it, starting on Monday. */
export const WEEK: readonly Day[] = [
  { id: 'mon', short: 'M', long: 'Monday' },
  { id: 'tue', short: 'T', long: 'Tuesday' },
  { id: 'wed', short: 'W', long: 'Wednesday' },
  { id: 'thu', short: 'T', long: 'Thursday' },
  { id: 'fri', short: 'F', long: 'Friday' },
  { id: 'sat', short: 'S', long: 'Saturday' },
  { id: 'sun', short: 'S', long: 'Sunday' },
]

/** The week turned to start on a day; a day it does not hold leaves it as it is. */
export const weekFrom = (id: string, week: readonly Day[] = WEEK): readonly Day[] => {
  const at = week.findIndex((day) => day.id === id)
  if (at <= 0) return week
  return [...week.slice(at), ...week.slice(0, at)]
}

/** What each day of the week carries, under the identifier of the day. */
export type Shares = Readonly<Record<string, number>>

/** The whole of a day, which a day nothing was said about carries. */
export const WHOLE = 100

/** The shares on offer, from nothing to the whole of a day. */
export const SHARES: readonly number[] = [0, 10, 25, 50, 75, 90, WHOLE]

/** What one day carries, which is the whole of it unless it says otherwise. */
export const shareOn = (shares: Shares, day: string): number => shares[day] ?? WHOLE

/**
 * The shares with one day put at a share, and a day back at the whole dropped:
 * what carries the whole of a day is what nothing was said about.
 */
export const shared = (shares: Shares, day: string, share: number): Shares => {
  const out: Record<string, number> = { ...shares }
  if (share === WHOLE) delete out[day]
  else out[day] = share
  return out
}
