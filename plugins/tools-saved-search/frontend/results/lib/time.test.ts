/**
 * The time a run finished, written as a date and a time in the locale of the person. The hour
 * depends on the zone of the machine, so the tests read the date and the shape of the time.
 */
import { describe, expect, it } from 'vitest'
import { formatTime } from './time'

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
