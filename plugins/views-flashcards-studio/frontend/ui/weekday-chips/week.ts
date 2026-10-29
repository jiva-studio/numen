/**
 * The seven days, with their display and full names.
 *
 * Pure: the rotation to a chosen first day of the week is arithmetic over a
 * list, and the names are supplied by the caller.
 */

/** What a day is called. */
export interface DayName {
  /** The caller's own identifier, handed back as given. */
  readonly id: string
  /** What is drawn on the chip: a letter or two. */
  readonly short: string
  /** The full name of the day, which is what a screen reader announces. */
  readonly long: string
}

/** One day as the row renders it: its name, and its fill level. */
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
export const rotateWeek = <One extends DayName>(
  id: string,
  week: readonly One[],
): readonly One[] => {
  const index = week.findIndex((day) => day.id === id)
  if (index <= 0) return week
  return [...week.slice(index), ...week.slice(0, index)]
}

/**
 * The fill of a chip, which is the level clamped to the range from empty to
 * full. A level outside that range cannot be displayed, and the figure shown and
 * the colour drawn are derived from this one number.
 */
export const getFill = (level: number): number => Math.min(Math.max(level, 0), 1)

/** The fill of a chip, as a percentage string. */
export const getFillPercent = (level: number): string => `${Math.round(getFill(level) * 100)}%`

/**
 * getMenuLevels returns the levels offered in a menu. A level in force that the
 * levels do not contain is added at its sorted position among them, so a day
 * always has its own level among the choices. When no level is in force the
 * levels are returned unchanged.
 */
export const getMenuLevels = (
  levels: readonly number[],
  inForce: number | null,
): readonly number[] => {
  if (inForce === null || levels.includes(inForce)) return levels
  const index = levels.findIndex((one) => one > inForce)
  if (index < 0) return [...levels, inForce]
  return [...levels.slice(0, index), inForce, ...levels.slice(index)]
}
