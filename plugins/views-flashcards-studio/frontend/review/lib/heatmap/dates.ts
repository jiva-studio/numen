/**
 * A day, formatted for the reader's locale.
 *
 * Nothing here contains a month name: the formatting comes from `Intl`, so a
 * date is displayed in the reader's own language.
 */

import { parseDay } from '../../../lib/day'

const full = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })

/** A day formatted as the reader's locale writes a date. */
export const dayName = (day: string): string => full.format(parseDay(day))
