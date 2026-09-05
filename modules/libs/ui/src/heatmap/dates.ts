/**
 * A day, in the words of whoever is reading.
 *
 * Nothing here holds a name for a month: the machine is asked, so a person
 * reading in their own language reads their own dates.
 */

import { dayOf } from '../calendar/day'

const full = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })

/** A day as a person reads one. */
export const dayName = (day: string): string => full.format(dayOf(day))
