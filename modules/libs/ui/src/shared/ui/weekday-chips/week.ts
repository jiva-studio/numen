/**
 * The seven days, as they are drawn and as they are named.
 *
 * Pure: which day the week is turned to start on is arithmetic over a list,
 * and the names are the caller's to hand in.
 */

/** What a day is called. */
export interface DayName {
  /** The caller's own identifier, handed back as given. */
  readonly id: string
  /** What is drawn on the chip: a letter or two. */
  readonly short: string
  /** What the day is called, which is what a screen reader says. */
  readonly long: string
}

/** One day as the row draws it: what it is called, and how full it stands. */
export interface Day extends DayName {
  /** How full the chip is drawn, from nothing to the whole of it. */
  readonly level: number
}

/** The week as English names it, starting on Monday. */
export const WEEK: readonly DayName[] = [
  { id: 'mon', short: 'M', long: 'Monday' },
  { id: 'tue', short: 'T', long: 'Tuesday' },
  { id: 'wed', short: 'W', long: 'Wednesday' },
  { id: 'thu', short: 'T', long: 'Thursday' },
  { id: 'fri', short: 'F', long: 'Friday' },
  { id: 'sat', short: 'S', long: 'Saturday' },
  { id: 'sun', short: 'S', long: 'Sunday' },
]

/** The week turned to start on a day; a day it does not hold leaves it as it is. */
export const weekFrom = <One extends DayName>(id: string, week: readonly One[]): readonly One[] => {
  const at = week.findIndex((day) => day.id === id)
  if (at <= 0) return week
  return [...week.slice(at), ...week.slice(0, at)]
}

/**
 * How full a chip is drawn, which is the level brought inside nothing and the
 * whole of it. A level outside that is none a chip can show, and the figure
 * said and the colour drawn are this one number.
 */
export const getFill = (level: number): number => Math.min(Math.max(level, 0), 1)

/** How full a chip stands, written out as a share of the whole. */
export const getFillPercent = (level: number): string => `${Math.round(getFill(level) * 100)}%`

/**
 * The levels on offer, holding the one a day stands at. A level the offer does
 * not name is added where it stands among them, so a day is never asked to
 * choose without its own level among the choices. No level in force leaves the
 * offer as it is.
 */
export const getOfferedLevels = (
  levels: readonly number[],
  inForce: number | null,
): readonly number[] => {
  if (inForce === null || levels.includes(inForce)) return levels
  const at = levels.findIndex((one) => one > inForce)
  if (at < 0) return [...levels, inForce]
  return [...levels.slice(0, at), inForce, ...levels.slice(at)]
}
