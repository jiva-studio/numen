/**
 * A day, in the words of whoever is reading.
 *
 * Nothing here holds a name for a month: the machine is asked, so a person
 * reading in their own language reads their own dates.
 */

const full = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })

/** A day as a person reads one. */
export const dayName = (day: string): string => {
  const [year, month, at] = day.split('-').map(Number)
  return full.format(new Date(year ?? 2000, (month ?? 1) - 1, at ?? 1))
}
