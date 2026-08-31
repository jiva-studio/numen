/**
 * The week as a list: where it is turned to start, and what each of its days
 * carries. Neither answer touches a chip.
 */
import { describe, expect, it } from 'vitest'
import { shared, shareOn, weekFrom, SHARES, WEEK, WHOLE } from './week'

const ids = (days: readonly { id: string }[]): readonly string[] => days.map((day) => day.id)

describe('where the week starts', () => {
  it('starts on Monday where it is asked for nothing else', () => {
    expect(ids(WEEK)).toEqual(['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'])
    expect(ids(weekFrom('mon'))).toEqual(ids(WEEK))
  })

  it('turns to the day it is given, keeping the days in order', () => {
    expect(ids(weekFrom('sun'))).toEqual(['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'])
    expect(ids(weekFrom('sat'))).toEqual(['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'])
  })

  it('leaves the week as it is where the day is not one of its own', () => {
    expect(ids(weekFrom('yesterday'))).toEqual(ids(WEEK))
  })
})

describe('what a day carries', () => {
  it('is the whole of a day for a day nothing was said about', () => {
    expect(shareOn({}, 'mon')).toBe(WHOLE)
    expect(shareOn({ sat: 50 }, 'mon')).toBe(WHOLE)
    expect(shareOn({ sat: 50 }, 'sat')).toBe(50)
  })

  it('is nothing where a day is put at nothing, which is not the whole of it', () => {
    expect(shareOn({ sun: 0 }, 'sun')).toBe(0)
  })

  // What carries the whole of a day is what nothing was said about, so a day
  // put back to it stops being named.
  it('names a day standing under the whole, and drops one put back to it', () => {
    expect(shared({}, 'sat', 50)).toEqual({ sat: 50 })
    expect(shared({ sat: 50 }, 'sat', 0)).toEqual({ sat: 0 })
    expect(shared({ sat: 50, sun: 0 }, 'sat', WHOLE)).toEqual({ sun: 0 })
    expect(shared({}, 'sat', WHOLE)).toEqual({})
  })

  it('leaves every other day where it stood', () => {
    expect(shared({ sat: 50, sun: 0 }, 'mon', 25)).toEqual({ sat: 50, sun: 0, mon: 25 })
  })

  it('offers the shares from nothing to the whole of a day', () => {
    expect(SHARES[0]).toBe(0)
    expect(SHARES.at(-1)).toBe(WHOLE)
    expect([...SHARES].sort((one, two) => one - two)).toEqual([...SHARES])
  })
})
