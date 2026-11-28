import { describe, expect, it } from 'vitest'
import { getTimeLeftText } from './expiry'

const NOW = Date.parse('2026-11-25T12:00:00Z')

describe('the time a link still works', () => {
  it('counts minutes below an hour, rounded up', () => {
    expect(getTimeLeftText('2026-11-25T12:09:10Z', NOW)).toBe('10 min')
  })

  it('counts hours from an hour on', () => {
    expect(getTimeLeftText('2026-11-25T14:00:00Z', NOW)).toBe('2 h')
  })

  it('shows 0 min once the link is over', () => {
    expect(getTimeLeftText('2026-11-25T11:00:00Z', NOW)).toBe('0 min')
  })
})
