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
})
