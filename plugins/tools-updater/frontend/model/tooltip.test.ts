import { describe, expect, it } from 'vitest'
import type { Update } from '../api/updater'
import { getTooltip } from './tooltip'

const UPDATE: Update = {
  version: '2026.10.1',
  build: 950,
  channel: 'stable',
  releasedAt: '2026-10-15T12:00:00Z',
  notesUrl: '',
  downloadUrl: 'https://dl.example.org/setup.exe',
}

describe('the tooltip of the version chip', () => {
  it('names both versions and the action when an update waits', () => {
    expect(getTooltip('2026.9.0', UPDATE, 'available')).toBe(
      'Update available: v2026.10.1. You run v2026.9.0. Click to download.',
    )
  })

  it('says the build is the latest once a check found no newer one', () => {
    expect(getTooltip('2026.9.0', null, 'current')).toBe(
      'Numen v2026.9.0. This is the latest version.',
    )
  })

  it('says only which version runs while nothing is known of a newer one', () => {
    expect(getTooltip('2026.9.0', null, 'unchecked')).toBe('Numen v2026.9.0, the version you run.')
  })
})
