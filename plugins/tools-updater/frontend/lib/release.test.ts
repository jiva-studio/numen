import { describe, expect, it } from 'vitest'
import { formatReleaseDate, getPlatformMessage } from './release'

describe('formatReleaseDate', () => {
  it('formats a valid ISO date', () => {
    expect(formatReleaseDate('2026-10-15T12:00:00Z', 'en-GB')).toBe('15 October 2026')
  })

  it('is empty for an invalid or missing date', () => {
    expect(formatReleaseDate('invalid-date')).toBe('')
    expect(formatReleaseDate(undefined)).toBe('')
    expect(formatReleaseDate('')).toBe('')
  })
})

describe('getPlatformMessage', () => {
  it('names the way to update on each platform', () => {
    expect(getPlatformMessage('windows')).toContain('installer')
    expect(getPlatformMessage('darwin')).toContain('disk image')
    expect(getPlatformMessage('linux')).toContain('package manager')
  })
})
