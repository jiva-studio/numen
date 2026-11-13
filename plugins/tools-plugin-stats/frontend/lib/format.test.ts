import { describe, expect, it } from 'vitest'
import { formatDuration, formatSummary } from './format'

const SECOND = 1_000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE

describe('a duration', () => {
  it('is written in its two largest units, the second one padded', () => {
    expect(formatDuration(45 * SECOND)).toBe('45s')
    expect(formatDuration(12 * MINUTE + 5 * SECOND)).toBe('12m 05s')
    expect(formatDuration(3 * HOUR + 7 * MINUTE + 59 * SECOND)).toBe('3h 07m')
    expect(formatDuration(52 * HOUR)).toBe('2d 04h')
  })

  it('drops the part of a second and writes a time past as zero', () => {
    expect(formatDuration(59_999)).toBe('59s')
    expect(formatDuration(-4 * SECOND)).toBe('0s')
  })
})

describe('the summary of the title bar item', () => {
  it('names the count of running plugins, or that none runs', () => {
    expect(formatSummary(0, 'en-US')).toBe('No plugin is running')
    expect(formatSummary(1, 'en-US')).toBe('1 plugin running')
    expect(formatSummary(3, 'en-US')).toBe('3 plugins running')
  })
})
