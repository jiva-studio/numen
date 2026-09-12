/**
 * A day of the calendar, written down and read back.
 *
 * A day travels as the year, the month and the day, and that is the only shape
 * arithmetic is done in here: two written days are counted apart on the
 * calendar itself, where an hour put into or taken out of a clock is not a
 * length of time at all. A day is read off an instant in the zone the machine
 * stands in, which is the calendar the person is looking at.
 */

/** How long a day of the calendar is, where no clock changes under it. */
const DAY = 86_400_000

/** A day as it is written down: the year, the month and the day. */
export const dayNamed = (at: Date): string => {
  const month = String(at.getMonth() + 1).padStart(2, '0')
  const day = String(at.getDate()).padStart(2, '0')
  return `${at.getFullYear()}-${month}-${day}`
}

/**
 * The instant a written day is read at, which is noon of it. A day is held away
 * from both its ends so that a clock moved an hour either way leaves it the day
 * it was.
 */
export const dayOf = (day: string): Date => {
  const [year, month, at] = day.split('-').map(Number)
  return new Date(year ?? 2000, (month ?? 1) - 1, at ?? 1, 12)
}

/** Whether a value is a day at all, which an empty field is not. */
export const isDay = (day: string): boolean => !Number.isNaN(Date.parse(`${day}T00:00:00Z`))

/**
 * How many days lie between two written days, and none where either is not a
 * day. A day before the other is a count below zero.
 */
export const daysBetween = (from: string, to: string): number => {
  if (!isDay(from) || !isDay(to)) return 0
  return Math.round((getDayNumber(to) - getDayNumber(from)) / DAY)
}

/** The day that many days after a written one, and itself where it is not a day. */
export const dayAfter = (from: string, days: number): string => {
  if (!isDay(from)) return from
  const at = new Date(getDayNumber(from))
  at.setUTCDate(at.getUTCDate() + days)
  return at.toISOString().slice(0, 10)
}

/**
 * A written day as a number of days, counted on a calendar no clock change
 * touches. It is not an instant and nothing is shown from it.
 */
const getDayNumber = (day: string): number => {
  const [year, month, at] = day.split('-').map(Number)
  return Date.UTC(year ?? 2000, (month ?? 1) - 1, at ?? 1)
}
