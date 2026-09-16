import { describe, expect, it } from 'vitest'
import { activity, percentWord, rateOf, getRemainingWord, shareOf, SMOOTHING } from './tally'

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
    expect(activity({ text: '' })).toEqual({ state: 'quiet', counts: false })
  })

  it('is quiet even with a tally, when it has no words', () => {
    expect(activity({ text: '', tally: { done: 1, total: 2 } })).toEqual({
      state: 'quiet',
      counts: false,
    })
  })

  it('rests when it has words and no work', () => {
    expect(activity({ text: 'searched by words alone' })).toEqual({
      state: 'resting',
      counts: false,
    })
  })

  it('works without a count when the work is claimed', () => {
    expect(activity({ text: 'reading the vault', isWorking: true })).toEqual({
      state: 'working',
      counts: false,
    })
  })

  it('works from a count alone, because a count that moves is work', () => {
    expect(activity({ text: 'reading', tally: { done: 1, total: 4 } })).toEqual({
      state: 'working',
      share: 0.25,
      counts: true,
    })
  })

  it('counts when there is a share to draw', () => {
    expect(activity({ text: 'reading', tally: { done: 2, total: 8 } })).toEqual({
      state: 'working',
      share: 0.25,
      counts: true,
    })
  })

  it('rests when a tally has no total to draw', () => {
    expect(activity({ text: 'reading', tally: { done: 0, total: 0 } })).toEqual({
      state: 'resting',
      counts: false,
    })
  })

  it('lets a failure outrank a count', () => {
    expect(activity({ text: 'reading', hasFailed: true, tally: { done: 2, total: 8 } })).toEqual({
      state: 'failed',
      counts: false,
    })
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

describe('getRemainingWord', () => {
  it.each([
    { left: 0, rate: 5, want: '', why: 'nothing left to wait for' },
    { left: 100, rate: 0, want: '', why: 'no rate is no estimate' },
    { left: 100, rate: -1, want: '' },
    { left: 30, rate: 5, want: '0:06' },
    { left: 600, rate: 5, want: '2:00' },
    { left: 300, rate: 5, want: '1:00' },
    { left: 36000, rate: 3, want: '3:20:00' },
    { left: 400000, rate: 3, want: '37:02:13' },
  ])('says $want for $left at $rate a second', ({ left, rate, want }) => {
    expect(getRemainingWord(left, rate)).toBe(want)
  })
})

describe('rateOf', () => {
  it('is what moved over the time it took, the first time', () => {
    expect(rateOf({ done: 0, rate: 0 }, 20, 2)).toBe(10)
  })

  it('leans on the rate already known when something moved', () => {
    expect(rateOf({ done: 20, rate: 10 }, 40, 2)).toBeCloseTo(10, 5)
    expect(rateOf({ done: 20, rate: 10 }, 60, 2)).toBeGreaterThan(10)
    expect(rateOf({ done: 20, rate: 10 }, 60, 2)).toBeLessThan(20)
  })

  it('leans further the longer the reading covers', () => {
    const brief = rateOf({ done: 20, rate: 10 }, 40, 1)
    const long = rateOf({ done: 20 * SMOOTHING, rate: 10 }, 40 * SMOOTHING, SMOOTHING)
    // Both saw twenty a second. The one that watched for a smoothing length
    // moved most of the way there; the one that watched for a second barely
    // moved.
    expect(brief).toBeLessThan(11)
    expect(long).toBeGreaterThan(15)
  })

  it('stands where it is when nothing moved, so no estimate grows out of it', () => {
    // A count written in groups stands still between them. Decaying the rate
    // here divides into a longer and longer estimate every time it is drawn.
    let rate = 8
    for (let stalled = 0; stalled < 40; stalled++) {
      rate = rateOf({ done: 100, rate }, 100, 2)
    }
    expect(rate).toBe(8)
  })

  it('keeps the old rate when no time has passed', () => {
    expect(rateOf({ done: 20, rate: 10 }, 40, 0)).toBe(10)
  })

  it('is nothing when the count went backwards, which is a fresh start', () => {
    expect(rateOf({ done: 100, rate: 10 }, 5, 2)).toBe(0)
  })
})
