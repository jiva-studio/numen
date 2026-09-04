import { describe, expect, it } from 'vitest'
import {
  activity,
  percentWord,
  rateOf,
  rateWord,
  remainingWord,
  shareOf,
  sizeWord,
  tallyWord,
} from './tally'

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
    // A vault loses a book mid-scan and the total falls below what was read.
    // The bar is already full, and the count has to agree with it.
    { done: 38, total: 37, want: '37 of 37' },
  ])('reads $done of $total as $want', ({ done, total, want }) => {
    expect(tallyWord({ done, total })).toBe(want)
  })

  it.each([
    { done: 0, total: 470_268_510, want: '0 B of 470 MB' },
    { done: 121_000_000, total: 470_268_510, want: '121 MB of 470 MB' },
    { done: 470_268_510, total: 470_268_510, want: '470 MB of 470 MB' },
  ])('counted in bytes, reads $done of $total as $want', ({ done, total, want }) => {
    expect(tallyWord({ done, total }, 'bytes')).toBe(want)
  })

  it.each([
    { done: 0, total: 5_400, want: '0:00 of 1:30:00' },
    { done: 95, total: 5_400, want: '1:35 of 1:30:00' },
    { done: 5_400, total: 5_400, want: '1:30:00 of 1:30:00' },
  ])('counted in seconds, reads $done of $total as $want', ({ done, total, want }) => {
    expect(tallyWord({ done, total }, 'seconds')).toBe(want)
  })
})

describe('sizeWord', () => {
  it.each([
    { bytes: 0, want: '0 B' },
    { bytes: 512, want: '512 B' },
    { bytes: 17_082_730, want: '17 MB' },
    { bytes: 470_268_510, want: '470 MB' },
    { bytes: 2_400_000_000, want: '2.4 GB' },
    { bytes: -1, want: '0 B' },
  ])('reads $bytes as $want', ({ bytes, want }) => {
    expect(sizeWord(bytes)).toBe(want)
  })
})

describe('rateWord', () => {
  it('says nothing about a rate nobody has measured', () => {
    expect(rateWord(0, 'bytes')).toBe('')
    expect(rateWord(-1, 'bytes')).toBe('')
  })

  it('reads a rate of bytes in the sizes a person reads', () => {
    expect(rateWord(12_400_000, 'bytes')).toBe('12 MB/s')
  })

  it('counts everything else one by one', () => {
    expect(rateWord(1420)).toBe('1 420/s')
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
    { left: 30, rate: 5, want: '0:06' },
    { left: 600, rate: 5, want: '2:00' },
    { left: 300, rate: 5, want: '1:00' },
    { left: 36000, rate: 3, want: '3:20:00' },
    { left: 400000, rate: 3, want: '37:02:13' },
  ])('says $want for $left at $rate a second', ({ left, rate, want }) => {
    expect(remainingWord(left, rate)).toBe(want)
  })
})

describe('rateOf', () => {
  it('is what moved over the time it took, the first time', () => {
    expect(rateOf({ done: 0, rate: 0 }, 20, 2)).toBe(10)
  })

  it('leans on the rate already known when something moved', () => {
    expect(rateOf({ done: 20, rate: 10 }, 40, 2)).toBeCloseTo(10, 5)
    expect(rateOf({ done: 20, rate: 10 }, 60, 2)).toBeCloseTo(13, 5)
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
