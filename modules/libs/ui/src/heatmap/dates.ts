/**
 * A day and a month, in the words of whoever is reading.
 *
 * Nothing here holds a name for a month: the machine is asked, so a person
 * reading in their own language reads their own months and their own dates.
 */
import type { Mark } from './heatmap'

/** One label over the grid, in words. */
export interface Named {
  readonly says: string
  readonly column: number
}

const month = new Intl.DateTimeFormat(undefined, { month: 'short' })
const withYear = new Intl.DateTimeFormat(undefined, { month: 'short', year: 'numeric' })
const full = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })

/** A day as a person reads one. */
export const dayName = (day: string): string => full.format(dated(day))

/** The months over the grid, each said where it opens. */
export const monthNames = (marks: readonly Mark[]): Named[] =>
  marks.map((one) => ({
    column: one.column,
    says: (one.year ? withYear : month).format(dated(one.day)),
  }))

/** A day, as the machine holds one. */
function dated(day: string): Date {
  const [year, month, at] = day.split('-').map(Number)
  return new Date(year ?? 2000, (month ?? 1) - 1, at ?? 1)
}
