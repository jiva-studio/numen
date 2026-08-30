/**
 * Where the one control's curve is drawn, and where a pointer over it stands.
 *
 * The picture is the control, so the same arithmetic is asked both ways: a
 * place put on the line, and a place read back off a pointer.
 */
import { describe, expect, it } from 'vitest'

import { NOWHERE, type Curve, type Point } from './core'
import {
  areaOf,
  bandOf,
  FOOT,
  LEFT,
  lineOf,
  MIDDLE,
  placeUnder,
  RIGHT,
  shortOf,
  spotsOf,
  TOP,
  WIDE,
  xOf,
  yOfBand,
} from './drawing'

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  met: true,
  ...over,
})

const curve = (retained: readonly number[], enough: readonly boolean[] = []): Curve => ({
  goal: 'minutes',
  grid: retained.map((_, at) => at * 10),
  days: [],
  at: retained.map((one, at) => point({ retained: one, enough: enough[at] ?? true })),
  now: NOWHERE,
  suggested: NOWHERE,
  decks: 1,
  honest: true,
})

describe('where a place of the grid stands across the picture', () => {
  it('puts the first at one edge of the room and the last at the other', () => {
    expect(xOf(0, 4)).toBe(LEFT)
    expect(xOf(3, 4)).toBe(RIGHT)
  })

  it('puts a single place at the edge it begins at', () => {
    expect(xOf(0, 1)).toBe(LEFT)
  })

  it('draws its faint lines between the foot and the top', () => {
    expect(yOfBand(0)).toBe(FOOT)
    expect(yOfBand(1)).toBe(TOP)
  })
})

describe('the curve as it is drawn', () => {
  it('is scaled to the band between the least and the most it costs', () => {
    const one = curve([0, 0.5, 1])
    expect(bandOf(one)).toStrictEqual({ least: 0, most: 1 })
    const spots = spotsOf(one)
    expect(spots[0]?.y).toBe(FOOT)
    expect(spots[2]?.y).toBe(TOP)
  })

  // A curve of a narrow band is the shape of that band, and not a line flat
  // against the top of a picture scaled to nothing.
  it('fills the picture with a band that moves little', () => {
    const spots = spotsOf(curve([0.9, 0.94, 0.98]))
    expect(spots[0]?.y).toBe(FOOT)
    expect(spots[2]?.y).toBe(TOP)
  })

  it('runs through the middle where it costs the same everywhere', () => {
    expect(spotsOf(curve([0, 0, 0])).every((spot) => spot.y === MIDDLE)).toBe(true)
    expect(spotsOf(curve([0.5, 0.5])).every((spot) => spot.y === MIDDLE)).toBe(true)
  })

  it('is one line through every place, closed down to the foot where it is filled', () => {
    const spots = spotsOf(curve([0, 0.5, 1]))
    expect(lineOf(spots).startsWith('M')).toBe(true)
    expect(lineOf(spots).split('L')).toHaveLength(3)
    expect(areaOf(spots).endsWith('Z')).toBe(true)
  })

  it('is nothing at all where the curve holds no place', () => {
    expect(lineOf([])).toBe('')
    expect(areaOf([])).toBe('')
  })
})

describe('the stretch a budget does not get through', () => {
  it('is a line of its own over the places the budget falls short at', () => {
    const one = curve([0, 0.4, 0.7, 1], [true, false, false, true])
    expect(shortOf(one, spotsOf(one)).startsWith('M')).toBe(true)
  })

  it('is nothing where the budget gets through all of it', () => {
    const one = curve([0, 0.4, 1])
    expect(shortOf(one, spotsOf(one))).toBe('')
  })

  it('is nothing where it falls short at one place alone, which draws no line', () => {
    const one = curve([0, 0.4, 1], [true, false, true])
    expect(shortOf(one, spotsOf(one))).toBe('')
  })
})

describe('the place a pointer stands over', () => {
  it('is the first at the left edge of the room and the last at the right', () => {
    expect(placeUnder(LEFT, 25)).toBe(0)
    expect(placeUnder(RIGHT, 25)).toBe(24)
  })

  it('is the nearer end for a pointer outside the room the line is drawn in', () => {
    expect(placeUnder(0, 25)).toBe(0)
    expect(placeUnder(WIDE, 25)).toBe(24)
  })

  it('reads back the place a spot was drawn at', () => {
    const spots = spotsOf(curve([0, 0.25, 0.5, 0.75, 1]))
    spots.forEach((spot, at) => expect(placeUnder(spot.x, spots.length)).toBe(at))
  })

  it('is the first place of a grid holding one', () => {
    expect(placeUnder(WIDE / 2, 1)).toBe(0)
  })
})
