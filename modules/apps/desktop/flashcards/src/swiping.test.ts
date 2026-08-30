import { describe, expect, it } from 'vitest'

import { FAR, STILL, drawn, swiped } from './swiping'

describe('swiped', () => {
  it('turns the card when the hand did not move', () => {
    expect(swiped({ moved: 0, open: false })).toEqual({ does: 'press' })
    expect(swiped({ moved: STILL, open: false })).toEqual({ does: 'press' })
    expect(swiped({ moved: -STILL, open: true })).toEqual({ does: 'press' })
  })

  it('brings the panel in when the card is taken far enough to the left', () => {
    expect(swiped({ moved: -FAR, open: false })).toEqual({ does: 'open' })
    expect(swiped({ moved: -FAR * 3, open: false })).toEqual({ does: 'open' })
  })

  it('sends the panel away when the card is taken far enough to the right', () => {
    expect(swiped({ moved: FAR, open: true })).toEqual({ does: 'shut' })
  })

  // A drag that started and stopped short leaves everything where it was, so a
  // hand that thought better of it asks for nothing.
  it('asks for nothing when the card was taken and not taken far enough', () => {
    expect(swiped({ moved: -FAR + 1, open: false })).toBeNull()
    expect(swiped({ moved: FAR - 1, open: true })).toBeNull()
  })

  // The panel is brought in from one side and sent away to the other, and the
  // same drag cannot do both.
  it('asks for nothing when the card is taken the way the panel already is', () => {
    expect(swiped({ moved: FAR * 3, open: false })).toBeNull()
    expect(swiped({ moved: -FAR * 3, open: true })).toBeNull()
  })
})

describe('drawn', () => {
  it('follows the hand from nothing to the whole of the panel', () => {
    expect(drawn({ moved: 0, open: false })).toBe(0)
    expect(drawn({ moved: -FAR / 2, open: false })).toBe(0.5)
    expect(drawn({ moved: -FAR, open: false })).toBe(1)
  })

  it('goes no further than the whole of it, and no less than none', () => {
    expect(drawn({ moved: -FAR * 5, open: false })).toBe(1)
    expect(drawn({ moved: FAR * 5, open: false })).toBe(0)
  })

  // With the panel up, the hand is taking it away, so nothing moved is the
  // whole of it and a drag to the right is what shrinks it.
  it('starts from the whole of it when the panel is already up', () => {
    expect(drawn({ moved: 0, open: true })).toBe(1)
    expect(drawn({ moved: FAR / 2, open: true })).toBe(0.5)
    expect(drawn({ moved: FAR, open: true })).toBe(0)
  })
})
