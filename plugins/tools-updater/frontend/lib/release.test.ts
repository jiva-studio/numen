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
  it('says how to update in plain words on each platform', () => {
    expect(getPlatformMessage('windows')).toBe('Download the new version to update Numen.')
    expect(getPlatformMessage('darwin')).toBe('Download the new version to update Numen.')
    expect(getPlatformMessage('linux')).toBe(
      'Download the new version, or update through your package manager.',
    )
  })
})
