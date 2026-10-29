/**
 * The week as a list: where it is turned to start, and how full a day is drawn.
 * Neither answer touches a chip.
 */
import { describe, expect, it } from 'vitest'
import { getFill, getFillPercent, getMenuLevels, rotateWeek, WEEK } from './week'

const ids = (days: readonly { id: string }[]): readonly string[] => days.map((day) => day.id)

describe('where the week starts', () => {
  it('starts on Monday where it is requested for nothing else', () => {
    expect(ids(WEEK)).toEqual(['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'])
    expect(ids(rotateWeek('mon', WEEK))).toEqual(ids(WEEK))
    expect(rotateWeek('mon', WEEK)).toBe(WEEK)
  })

  it('turns to the day it is given, keeping the days in order', () => {
    expect(ids(rotateWeek('sun', WEEK))).toEqual(['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'])
    expect(ids(rotateWeek('sat', WEEK))).toEqual(['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'])
  })

  it('leaves the week as it is where the day is not one of its own', () => {
    expect(ids(rotateWeek('yesterday', WEEK))).toEqual(ids(WEEK))
  })
})

describe('how full a day is drawn', () => {
  it('is the level it remains at, where that is one a chip can show', () => {
    expect(getFill(0.37)).toBe(0.37)
    expect(getFill(0)).toBe(0)
    expect(getFill(1)).toBe(1)
  })

  // The figure said and the colour drawn are one number, so a level the row
  // cannot draw is not a level it announces either.
  it('is brought inside nothing and the whole at its current position outside them', () => {
    expect(getFill(4)).toBe(1)
    expect(getFill(-0.2)).toBe(0)
  })

  it('is written out as a share of the whole', () => {
    expect(getFillPercent(0.5)).toBe('50%')
    expect(getFillPercent(1)).toBe('100%')
    expect(getFillPercent(0)).toBe('0%')
    expect(getFillPercent(0.37)).toBe('37%')
    expect(getFillPercent(4)).toBe('100%')
  })
})

describe('what a day is offered', () => {
  const LEVELS: readonly number[] = [0, 0.25, 0.5, 1]

  it('is the levels as they were given, where its own is among them', () => {
    expect(getMenuLevels(LEVELS, 0.5)).toEqual(LEVELS)
    expect(getMenuLevels(LEVELS, null)).toEqual(LEVELS)
  })

  it('holds the level it remains at, in its place among them', () => {
    expect(getMenuLevels(LEVELS, 0.37)).toEqual([0, 0.25, 0.37, 0.5, 1])
    expect(getMenuLevels([0, 0.25, 0.5], 0.9)).toEqual([0, 0.25, 0.5, 0.9])
    expect(getMenuLevels([0.25, 0.5], 0)).toEqual([0, 0.25, 0.5])
  })
})
