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
