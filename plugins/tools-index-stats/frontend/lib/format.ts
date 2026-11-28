/**
 * How the numbers of the index are written for a person: counts with the thousands separator of the
 * locale, sizes in the largest unit that keeps the figure above one, and the time of a change.
 */
import type { Stats } from '../types'
import { WORDS as words } from '../words'
import { isPending } from './pending'

/** The mark in front of the count in the title bar. */
export const MARK = '⬡'

/** A whole number with the thousands separator of the locale; the locale of the window when none. */
export function formatCount(value: number, locale?: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value)
}

/** The figure of the title bar item: the vectors the index holds, and never the chunks. */
export function formatVectors(stats: Pick<Stats, 'vectors'>, locale?: string): string {
  return formatCount(stats.vectors, locale)
}

/** The tooltip and the accessible name of the item: the vectors, and that indexing runs. */
export function formatSummary(stats: Pick<Stats, 'vectors' | 'waiting'>, locale?: string): string {
  const vectors = `${formatCount(stats.vectors, locale)} ${words.vectorsUnit}`
  return isPending(stats) ? `${vectors}, ${words.indexing}` : vectors
}

/** A rate with one decimal. */
export function formatRate(perSecond: number, locale?: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(perSecond)
}

/** The time of day of an instant given in RFC 3339, and an empty text for one that is not. */
export function formatTime(instant: string, locale?: string): string {
  const moment = new Date(instant)
  if (Number.isNaN(moment.getTime())) return ''
  return new Intl.DateTimeFormat(locale, {
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
  }).format(moment)
}
