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

/** The days that are on, in the order the week is drawn in. */
export const lit = (chosen: readonly string[], week: readonly Day[] = WEEK): readonly string[] => {
  const among = new Set(chosen)
  return week.filter((day) => among.has(day.id)).map((day) => day.id)
}
