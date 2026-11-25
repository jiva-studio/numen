import { describe, expect, it } from 'vitest'
import { formatLastSync } from './lastSync'

const NOW = new Date('2026-11-25T12:00:00Z')
const MINUTE = 60_000
const ago = (minutes: number): Date => new Date(NOW.getTime() - minutes * MINUTE)

describe('formatLastSync', () => {
  it.each([
    [0, 'just now'],
    [-5, 'just now'],
    [1, 'a minute ago'],
    [12, '12 minutes ago'],
    [59, '59 minutes ago'],
    [60, 'about an hour ago'],
    [237, 'about 3 hours ago'],
    [23 * 60 + 59, 'about 23 hours ago'],
    [24 * 60, 'yesterday'],
    [47 * 60, 'yesterday'],
    [48 * 60, '2 days ago'],
    [6 * 24 * 60 + 100, '6 days ago'],
  ])('says %i minutes as "%s"', (minutes, text) => {
    expect(formatLastSync(ago(minutes), NOW)).toBe(text)
  })

  it('gives a date from a week on', () => {
    expect(formatLastSync(new Date('2026-11-10T12:00:00Z'), NOW)).toBe('Nov 10, 2026')
  })
})
