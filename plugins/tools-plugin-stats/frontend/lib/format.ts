/**
 * How the times of a running plugin are written for a person: a duration in its two largest units,
 * and the count of running plugins.
 */
import { WORDS as words } from '../words'

/** The mark in front of the count in the title bar. */
export const MARK = '◈'

const SECOND = 1_000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const pad = (value: number) => String(value).padStart(2, '0')

/**
 * A duration in its two largest units, the second one padded: `45s`, `12m 05s`, `3h 07m`, `2d 04h`.
 * A negative duration is written as zero.
 */
export function formatDuration(ms: number): string {
  const total = Math.max(0, Math.floor(ms / SECOND)) * SECOND
  if (total < MINUTE) return `${total / SECOND}s`
  if (total < HOUR)
    return `${Math.floor(total / MINUTE)}m ${pad(Math.floor((total % MINUTE) / SECOND))}s`
  if (total < DAY)
    return `${Math.floor(total / HOUR)}h ${pad(Math.floor((total % HOUR) / MINUTE))}m`
  return `${Math.floor(total / DAY)}d ${pad(Math.floor((total % DAY) / HOUR))}h`
}

/** A whole number with the thousands separator of the locale; the locale of the window when none. */
export function formatCount(value: number, locale?: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value)
}

/** The tooltip and the accessible name of the title bar item: how many plugins run. */
export function formatSummary(count: number, locale?: string): string {
  if (count === 0) return words.none
  return `${formatCount(count, locale)} ${count === 1 ? words.pluginRunning : words.pluginsRunning}`
}
