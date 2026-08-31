/**
 * Where the one control's curve is drawn, and where a pointer over it stands.
 *
 * The picture is the control, so the same arithmetic is asked both ways: a
 * place put on the line, and a place read back off a pointer.
 */
import { describe, expect, it } from 'vitest'

import { NOWHERE, type Curve, type Point } from './core'
import {
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
  clears: 0,
  ...over,
})

/** A curve of a goal of minutes, which is read in the cards a day answers. */
const curve = (cards: readonly number[], enough: readonly boolean[] = []): Curve => ({
  goal: 'minutes',
  grid: cards.map((_, at) => at * 10),
  days: [],
  at: cards.map((one, at) => point({ reviews: one, enough: enough[at] ?? true })),
  now: NOWHERE,
  suggested: NOWHERE,
  decks: 1,
  cards: 400,
  overdue: 0,
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
  it('is scaled to the band it is given, between the foot and the top', () => {
    const one = curve([0, 0.5, 1])
    expect(bandOf(one)).toStrictEqual({ least: 0, most: 1 })
    const spots = spotsOf(one, bandOf(one))
    expect(spots[0]?.y).toBe(FOOT)
    expect(spots[2]?.y).toBe(TOP)
  })

  // The band is the goal's and not this answer's, so an answer that moves
  // little inside it is drawn as the little it moves.
  it('draws a curve that is flat within its band flat', () => {
    const spots = spotsOf(curve([0.9, 0.9, 0.9]), { least: 0, most: 1 })
    expect(spots.every((spot) => spot.y === spots[0]?.y)).toBe(true)
    expect(spots[0]?.y).not.toBe(MIDDLE)
  })

  it('keeps a curve past either end of its band inside the picture', () => {
    const spots = spotsOf(curve([-1, 0.5, 2]), { least: 0, most: 1 })
    expect(spots[0]?.y).toBe(FOOT)
    expect(spots[2]?.y).toBe(TOP)
  })

  it('runs through the middle where the band has no width', () => {
    expect(spotsOf(curve([0, 0, 0]), { least: 0, most: 0 }).every((one) => one.y === MIDDLE)).toBe(
      true,
    )
  })

  it('is one line through every place', () => {
    const spots = spotsOf(curve([0, 0.5, 1]), { least: 0, most: 1 })
    expect(lineOf(spots).startsWith('M')).toBe(true)
    expect(lineOf(spots).split('L')).toHaveLength(3)
  })

  it('is nothing at all where the curve holds no place', () => {
    expect(lineOf([])).toBe('')
  })
})

describe('the stretch a budget does not get through', () => {
  it('is a line of its own over the places the budget falls short at', () => {
    const one = curve([0, 0.4, 0.7, 1], [true, false, false, true])
    expect(shortOf(one, spotsOf(one, bandOf(one))).startsWith('M')).toBe(true)
  })

  it('is nothing where the budget gets through all of it', () => {
    const one = curve([0, 0.4, 1])
    expect(shortOf(one, spotsOf(one, bandOf(one)))).toBe('')
  })

  it('is nothing where it falls short at one place alone, which draws no line', () => {
    const one = curve([0, 0.4, 1], [true, false, true])
    expect(shortOf(one, spotsOf(one, bandOf(one)))).toBe('')
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
    const one = curve([0, 0.25, 0.5, 0.75, 1])
    const spots = spotsOf(one, bandOf(one))
    spots.forEach((spot, at) => expect(placeUnder(spot.x, spots.length)).toBe(at))
  })

  it('is the first place of a grid holding one', () => {
    expect(placeUnder(WIDE / 2, 1)).toBe(0)
  })
})
