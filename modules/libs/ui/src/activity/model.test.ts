import { describe, expect, it } from 'vitest'
import {
  activity,
  percentWord,
  rateOf,
  remainingWord,
  shareOf,
  tallyWord,
} from './model'

describe('shareOf', () => {
  it.each([
    { done: 0, total: 0, want: undefined, why: 'nothing to do has no share' },
    { done: 0, total: 10, want: 0, why: 'nothing done' },
    { done: 5, total: 10, want: 0.5, why: 'half' },
    { done: 10, total: 10, want: 1, why: 'all of it' },
    { done: 12, total: 10, want: 1, why: 'a tally that overtook its total reads as full' },
    { done: -1, total: 10, want: 0, why: 'a count below nothing is nothing' },
    { done: 1, total: -3, want: undefined, why: 'a total below nothing is no total' },
  ])('$why', ({ done, total, want }) => {
    expect(shareOf({ done, total })).toBe(want)
  })
})

describe('activity', () => {
  it('says nothing when there is nothing to say', () => {
    expect(activity({ says: '' })).toEqual({ state: 'quiet', counts: false })
  })

  it('is quiet even with a tally, when it has no words', () => {
    expect(activity({ says: '', tally: { done: 1, total: 2 } })).toEqual({
      state: 'quiet',
      counts: false,
    })
  })

  it('rests when it has words and no work', () => {
    expect(activity({ says: 'searched by words alone' })).toEqual({
      state: 'resting',
      counts: false,
    })
  })

  it('works without a count when the work is claimed', () => {
    expect(activity({ says: 'reading the vault', working: true })).toEqual({
      state: 'working',
      counts: false,
    })
  })

  it('works from a count alone, because a count that moves is work', () => {
    expect(activity({ says: 'reading', tally: { done: 1, total: 4 } })).toEqual({
      state: 'working',
      share: 0.25,
      counts: true,
    })
  })

  it('counts when there is a share to draw', () => {
    expect(activity({ says: 'reading', tally: { done: 2, total: 8 } })).toEqual({
      state: 'working',
      share: 0.25,
      counts: true,
    })
  })

  it('rests when a tally has no total to draw', () => {
    expect(activity({ says: 'reading', tally: { done: 0, total: 0 } })).toEqual({
      state: 'resting',
      counts: false,
    })
  })

  it('lets trouble outrank a count', () => {
    expect(
      activity({ says: 'reading', trouble: true, tally: { done: 2, total: 8 } }),
    ).toEqual({ state: 'trouble', counts: false })
  })
})

describe('tallyWord', () => {
  it.each([
    { done: 0, total: 0, want: '0 of 0' },
    { done: 7, total: 40, want: '7 of 40' },
    { done: 1200, total: 36560, want: '1 200 of 36 560' },
    { done: 145800, total: 145800, want: '145 800 of 145 800' },
    { done: -5, total: 10, want: '0 of 10' },
  ])('reads $done of $total as $want', ({ done, total, want }) => {
    expect(tallyWord({ done, total })).toBe(want)
  })
})

describe('percentWord', () => {
  it.each([
    { share: 0, want: '0%' },
    { share: 0.004, want: '0%' },
    { share: 0.25, want: '25%' },
    { share: 0.999, want: '99%', why: 'nothing reads as finished until it is' },
    { share: 1, want: '100%' },
    { share: -1, want: '0%' },
  ])('reads $share as $want', ({ share, want }) => {
    expect(percentWord(share)).toBe(want)
  })
})

describe('remainingWord', () => {
  it.each([
    { left: 0, rate: 5, want: '', why: 'nothing left to wait for' },
    { left: 100, rate: 0, want: '', why: 'no rate is no estimate' },
    { left: 100, rate: -1, want: '' },
    { left: 30, rate: 5, want: 'under a minute left' },
    { left: 600, rate: 5, want: 'about 2 minutes left' },
    { left: 300, rate: 5, want: 'about 1 minute left' },
    { left: 36000, rate: 3, want: 'about 3 hours left' },
    { left: 400000, rate: 3, want: 'about 2 days left' },
  ])('says $want for $left at $rate a second', ({ left, rate, want }) => {
    expect(remainingWord(left, rate)).toBe(want)
  })
})

describe('rateOf', () => {
  it('is what moved over the time it took, the first time', () => {
    expect(rateOf({ done: 0, rate: 0 }, 20, 2)).toBe(10)
  })

  it('leans on the rate already known', () => {
    // A group of nothing between two readings keeps the rate already known.
    expect(rateOf({ done: 20, rate: 10 }, 20, 2)).toBeCloseTo(7, 5)
  })

  it('keeps the old rate when no time has passed', () => {
    expect(rateOf({ done: 20, rate: 10 }, 40, 0)).toBe(10)
  })

  it('is nothing when the count went backwards, which is a fresh start', () => {
    expect(rateOf({ done: 100, rate: 10 }, 5, 2)).toBe(0)
  })
})
