import { describe, expect, it } from 'vitest'

import { getTimeAhead } from './cards'

describe('how long a card is away for', () => {
  it('is read at the coarsest a person reads it by', () => {
    expect(getTimeAhead(30)).toBe('1m')
    expect(getTimeAhead(11 * 60)).toBe('11m')
    expect(getTimeAhead(90 * 60)).toBe('2h')
    expect(getTimeAhead(3 * 24 * 3600)).toBe('3d')
    expect(getTimeAhead(60 * 24 * 3600)).toBe('2mo')
    expect(getTimeAhead(3 * 365 * 24 * 3600)).toBe('3y')
  })

  // A card coming round in seconds is a card coming round now, and there is no
  // smaller word to say it in.
  it('is never less than a minute', () => {
    expect(getTimeAhead(0)).toBe('1m')
    expect(getTimeAhead(5)).toBe('1m')
  })

  // Sixty minutes is an hour: each unit gives way to the next at the length the
  // next one begins at.
  it('moves to the next unit at the length it begins at', () => {
    expect(getTimeAhead(59 * 60)).toBe('59m')
    expect(getTimeAhead(60 * 60)).toBe('1h')
    expect(getTimeAhead(23 * 3600)).toBe('23h')
    expect(getTimeAhead(24 * 3600)).toBe('1d')
    expect(getTimeAhead(29 * 24 * 3600)).toBe('29d')
    expect(getTimeAhead(30 * 24 * 3600)).toBe('1mo')
    expect(getTimeAhead(330 * 24 * 3600)).toBe('11mo')
    expect(getTimeAhead(360 * 24 * 3600)).toBe('1y')
  })
})
