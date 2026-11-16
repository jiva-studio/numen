/**
 * The time of a run, written as a date and a time in the locale of the person. The date and the time are
 * formatted apart and joined with a comma, so every engine writes the same text: WebKit joins the two
 * styles of one formatter with "at".
 */

/** The medium date and the short time of the locale, as `14 Nov 2026, 09:30`; '' for no time. */
export function formatTime(iso: string, locale: string): string {
  if (iso === '') return ''
  const at = new Date(iso)
  if (Number.isNaN(at.getTime())) return ''
  const date = new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(at)
  const time = new Intl.DateTimeFormat(locale, { timeStyle: 'short' }).format(at)
  return `${date}, ${time}`
}

const UNITS: readonly (readonly [Intl.RelativeTimeFormatUnit, number])[] = [
  ['second', 60],
  ['minute', 60],
  ['hour', 24],
]

/**
 * The time of a run as it reads from now, as `2 minutes ago`, within a day of now; the date and the
 * time further back or ahead; '' for no time.
 */
export function formatRelativeTime(iso: string, now: number, locale: string): string {
  if (iso === '') return ''
  const at = Date.parse(iso)
  if (Number.isNaN(at)) return ''
  const format = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  let amount = Math.round((at - now) / 1000)
  for (const [unit, size] of UNITS) {
    if (Math.abs(amount) < size) return format.format(amount, unit)
    amount = Math.round(amount / size)
  }
  return formatTime(iso, locale)
}
