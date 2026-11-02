/**
 * How the numbers of the index are written for a person: counts with the thousands separator of the
 * locale, sizes in the largest unit that keeps the figure above one, and the time of a change.
 */
import type { Stats } from '../types'

/** The mark in front of the count in the status bar. */
export const MARK = '⬡'

const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const
const BYTES_PER_UNIT = 1024

/** A whole number with the thousands separator of the locale; the locale of the window when none. */
export function formatCount(value: number, locale?: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value)
}

/**
 * The counts of the status bar item: the vectors the index holds, and while chunks wait for theirs,
 * how many chunks should carry one.
 */
export function formatCounts(
  stats: Pick<Stats, 'vectors' | 'chunks' | 'waiting'>,
  locale?: string,
): string {
  const vectors = formatCount(stats.vectors, locale)
  if (stats.waiting <= 0) return vectors
  return `${vectors} / ${formatCount(stats.chunks, locale)}`
}

/** The text of the status bar item: the mark and the counts. */
export function formatLabel(
  stats: Pick<Stats, 'vectors' | 'chunks' | 'waiting'>,
  locale?: string,
): string {
  return `${MARK} ${formatCounts(stats, locale)}`
}

/** A size in bytes in the largest unit that keeps the figure at one or more, with one decimal. */
export function formatBytes(bytes: number, locale?: string): string {
  let size = Math.max(0, bytes)
  let unit = 0
  while (size >= BYTES_PER_UNIT && unit < BYTE_UNITS.length - 1) {
    size /= BYTES_PER_UNIT
    unit += 1
  }
  const digits = unit === 0 ? 0 : 1
  const figure = new Intl.NumberFormat(locale, {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(size)
  return `${figure} ${BYTE_UNITS[unit]}`
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
