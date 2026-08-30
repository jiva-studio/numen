/**
 * The week as a list: where it is turned to start, and which of its days are
 * on. Neither answer touches a chip.
 */
import { describe, expect, it } from 'vitest'
import { lit, weekFrom, WEEK } from './week'

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

describe('which days are on', () => {
  it('says them in the order the week is drawn in', () => {
    expect(lit(['sun', 'tue'])).toEqual(['tue', 'sun'])
    expect(lit(['sun', 'tue'], weekFrom('sun'))).toEqual(['sun', 'tue'])
  })

  it('drops a day the week does not hold, and says each of them once', () => {
    expect(lit(['sat', 'yesterday', 'sat'])).toEqual(['sat'])
    expect(lit([])).toEqual([])
  })
})
