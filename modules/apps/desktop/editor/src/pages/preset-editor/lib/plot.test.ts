/**
 * Where the one control's curve is drawn, and where a pointer over it stands.
 *
 * The picture is the control, so the same arithmetic is asked both ways: a
 * place put on the line, and a place read back off a pointer.
 */
import { describe, expect, it } from 'vitest'

import { NOWHERE, type Curve, type Point } from '../types'
import {
  extentOf,
  FOOT,
  LEFT,
  lineOf,
  placeUnder,
  positionsOf,
  RIGHT,
  shortOf,
  TOP,
  walkGrid,
  WIDE,
  xOf,
  yOfGridline,
} from './plot'

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
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
  unbegun: 0,
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
    expect(yOfGridline(0)).toBe(FOOT)
    expect(yOfGridline(1)).toBe(TOP)
  })
})

describe('the curve as it is drawn', () => {
  it('is scaled to the extent it is given, between the foot and the top', () => {
    const one = curve([0, 0.5, 1])
    expect(extentOf(one)).toStrictEqual({ least: 0, most: 1 })
    const positions = positionsOf(one, extentOf(one))
    expect(positions[0]?.y).toBe(FOOT)
    expect(positions[2]?.y).toBe(TOP)
  })

  // The extent is the goal's and not this answer's, so an answer that moves
  // little inside it is drawn as the little it moves.
  it('draws a curve that is flat within its extent flat', () => {
    const positions = positionsOf(curve([0.9, 0.9, 0.9]), { least: 0, most: 1 })
    expect(positions.every((at) => at.y === positions[0]?.y)).toBe(true)
  })

  it('keeps a curve past either end of its extent inside the picture', () => {
    const positions = positionsOf(curve([-1, 0.5, 2]), { least: 0, most: 1 })
    expect(positions[0]?.y).toBe(FOOT)
    expect(positions[2]?.y).toBe(TOP)
  })

  // A count does not go below nothing, so nothing is the foot of the picture
  // and a run of it lies along that foot.
  it('lies along the floor where the extent has no width', () => {
    const positions = positionsOf(curve([0, 0, 0]), { least: 0, most: 0 })
    expect(positions.every((one) => one.y === FOOT)).toBe(true)
  })

  it('stands an extent on nothing, whatever the run it holds comes to', () => {
    expect(extentOf(curve([0, 0, 0]))).toStrictEqual({ least: 0, most: 0 })
    expect(extentOf(curve([40, 45, 41]))).toStrictEqual({ least: 0, most: 45 })
    expect(extentOf(curve([0, 20, 5]))).toStrictEqual({ least: 0, most: 20 })
    expect(extentOf(curve([]))).toStrictEqual({ least: 0, most: 0 })
  })

  // Whatever the run holds, the height read as nothing is the foot line.
  it('draws nothing on the foot under every run', () => {
    for (const run of [[0, 0, 0], [0, 20, 5], [40, 45, 41]]) {
      const one = curve(run)
      const positions = positionsOf(one, extentOf(one))
      for (const [at, value] of run.entries()) {
        if (value === 0) expect(positions[at]?.y).toBe(FOOT)
        expect(positions[at]?.y).toBeLessThanOrEqual(FOOT)
      }
    }
  })

  it('is one line through every place', () => {
    const positions = positionsOf(curve([0, 0.5, 1]), { least: 0, most: 1 })
    expect(lineOf(positions).startsWith('M')).toBe(true)
    expect(lineOf(positions).split('L')).toHaveLength(3)
  })

  it('is nothing at all where the curve holds no place', () => {
    expect(lineOf([])).toBe('')
  })
})

describe('the stretch a budget does not get through', () => {
  it('is a line of its own over the places the budget falls short at', () => {
    const one = curve([0, 0.4, 0.7, 1], [true, false, false, true])
    expect(shortOf(one, positionsOf(one, extentOf(one))).startsWith('M')).toBe(true)
  })

  it('is nothing where the budget gets through all of it', () => {
    const one = curve([0, 0.4, 1])
    expect(shortOf(one, positionsOf(one, extentOf(one)))).toBe('')
  })

  it('is nothing where it falls short at one place alone, which draws no line', () => {
    const one = curve([0, 0.4, 1], [true, false, true])
    expect(shortOf(one, positionsOf(one, extentOf(one)))).toBe('')
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

  it('reads back the place a point was drawn at', () => {
    const one = curve([0, 0.25, 0.5, 0.75, 1])
    const positions = positionsOf(one, extentOf(one))
    positions.forEach((at, place) => expect(placeUnder(at.x, positions.length)).toBe(place))
  })

  it('is the first place of a grid holding one', () => {
    expect(placeUnder(WIDE / 2, 1)).toBe(0)
  })
})

describe('where a keystroke takes the knob', () => {
  it('is one place either way, and stops at either end', () => {
    expect(walkGrid('ArrowRight', 2, 5)).toBe(3)
    expect(walkGrid('ArrowLeft', 2, 5)).toBe(1)
    expect(walkGrid('ArrowLeft', 0, 5)).toBe(0)
    expect(walkGrid('ArrowRight', 4, 5)).toBe(4)
  })

  it('is the same either way for the two axes, so a knob answers both', () => {
    expect(walkGrid('ArrowDown', 2, 5)).toBe(walkGrid('ArrowLeft', 2, 5))
    expect(walkGrid('ArrowUp', 2, 5)).toBe(walkGrid('ArrowRight', 2, 5))
  })

  it('is either end of the range', () => {
    expect(walkGrid('Home', 2, 5)).toBe(0)
    expect(walkGrid('End', 2, 5)).toBe(4)
  })

  it('is nothing for a keystroke of somebody else’s', () => {
    expect(walkGrid('a', 2, 5)).toBeNull()
    expect(walkGrid('Enter', 2, 5)).toBeNull()
  })
})
