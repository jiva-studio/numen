/**
 * The time a run finished, written as a date and a time in the locale of the person. The hour
 * depends on the zone of the machine, so the tests read the date and the shape of the time.
 */
import { describe, expect, it } from 'vitest'
import { formatRelativeTime, formatTime } from './time'

describe('formatTime', () => {
  it('formats the time in the locale given', () => {
    expect(formatTime('2026-11-14T12:00:00Z', 'en-GB')).toMatch(/^14 Nov 2026, \d\d:\d\d$/)
    expect(formatTime('2026-11-14T12:00:00Z', 'de-DE')).toMatch(/^14\.11\.2026, \d\d:\d\d$/)
  })

  it('writes the same instant the same in every call', () => {
    expect(formatTime('2026-11-14T12:00:00Z', 'en-GB')).toBe(
      formatTime('2026-11-14T12:00:00.000Z', 'en-GB'),
    )
  })

  it('writes nothing for no time', () => {
    expect(formatTime('', 'en-GB')).toBe('')
  })
})

describe('formatRelativeTime', () => {
  const NOW = Date.parse('2026-11-14T12:00:00Z')

  it('writes a time within a day as it reads from now', () => {
    expect(formatRelativeTime('2026-11-14T12:00:00Z', NOW, 'en-GB')).toBe('now')
    expect(formatRelativeTime('2026-11-14T11:59:30Z', NOW, 'en-GB')).toBe('30 seconds ago')
    expect(formatRelativeTime('2026-11-14T11:58:00Z', NOW, 'en-GB')).toBe('2 minutes ago')
    expect(formatRelativeTime('2026-11-14T09:00:00Z', NOW, 'en-GB')).toBe('3 hours ago')
  })

  it('writes the date and the time of a run more than a day back', () => {
    expect(formatRelativeTime('2026-11-12T12:00:00Z', NOW, 'en-GB')).toMatch(
      /^12 Nov 2026, \d\d:\d\d$/,
    )
  })

  it('writes nothing for no time', () => {
    expect(formatRelativeTime('', NOW, 'en-GB')).toBe('')
  })
})
