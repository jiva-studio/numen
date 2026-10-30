/**
 * Where the one control's curve is drawn, and where a pointer over it is positioned.
 *
 * The plot is the control, so the same arithmetic runs in both directions: a
 * place put on the line, and a place read back from a pointer.
 */
import { describe, expect, it } from 'vitest'

import { NO_CURVE_POINT, type Curve, type Point } from '../types'
import {
  getExtent,
  getBacklogExtent,
  PLOT_BOTTOM,
  PLOT_LEFT,
  createLinePath,
  getPlaceAtX,
  createPositions,
  PLOT_RIGHT,
  getBacklogRun,
  createSeries,
  getShortPath,
  PLOT_TOP,
  getKeyPlace,
  PLOT_WIDTH,
  getXAtPlace,
  yOfGridline,
} from './plot'

const point = (overrides: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  due: 0,
  through: 0,
  canLearnEveryCard: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
  ...overrides,
})

/** A curve of a goal of minutes, which is read in the cards a day answers. */
const curve = (cards: readonly number[], enough: readonly boolean[] = []): Curve => ({
  goal: 'minutes',
  grid: cards.map((_, index) => index * 10),
  days: [],
  at: cards.map((one, index) => point({ reviews: one, canLearnEveryCard: enough[index] ?? true })),
  now: NO_CURVE_POINT,
  suggested: NO_CURVE_POINT,
  decks: 1,
  cards: 400,
  overdue: 0,
  unbegun: 0,
  isHonest: true,
})

describe('where a place of the grid remains across the picture', () => {
  it('puts the first at one edge of the room and the last at the other', () => {
    expect(getXAtPlace(0, 4)).toBe(PLOT_LEFT)
    expect(getXAtPlace(3, 4)).toBe(PLOT_RIGHT)
  })

  it('puts a single place at the edge it begins at', () => {
    expect(getXAtPlace(0, 1)).toBe(PLOT_LEFT)
  })

  it('draws its faint lines between the foot and the top', () => {
    expect(yOfGridline(0)).toBe(PLOT_BOTTOM)
    expect(yOfGridline(1)).toBe(PLOT_TOP)
  })
})

describe('the curve as it is drawn', () => {
  it('is scaled to the extent it is given, between the foot and the top', () => {
    const one = curve([0, 0.5, 1])
    expect(getExtent(one)).toStrictEqual({ min: 0, max: 1 })
    const positions = createPositions(one, getExtent(one))
    expect(positions[0]?.y).toBe(PLOT_BOTTOM)
    expect(positions[2]?.y).toBe(PLOT_TOP)
  })

  // The extent is the goal's and not this answer's, so an answer that moves
  // little inside it is drawn as the little it moves.
  it('draws a curve that is flat within its extent flat', () => {
    const positions = createPositions(curve([0.9, 0.9, 0.9]), { min: 0, max: 1 })
    expect(positions.every((point) => point.y === positions[0]?.y)).toBe(true)
  })

  it('keeps a curve past either end of its extent inside the picture', () => {
    const positions = createPositions(curve([-1, 0.5, 2]), { min: 0, max: 1 })
    expect(positions[0]?.y).toBe(PLOT_BOTTOM)
    expect(positions[2]?.y).toBe(PLOT_TOP)
  })

  // A count is never below zero, so zero is the foot of the plot and a run of
  // zeros is drawn along that foot.
  it('lies along the floor where the extent has no width', () => {
    const positions = createPositions(curve([0, 0, 0]), { min: 0, max: 0 })
    expect(positions.every((one) => one.y === PLOT_BOTTOM)).toBe(true)
  })

  it('remains an extent on nothing, whatever the run it holds evaluates to', () => {
    expect(getExtent(curve([0, 0, 0]))).toStrictEqual({ min: 0, max: 0 })
    expect(getExtent(curve([40, 45, 41]))).toStrictEqual({ min: 0, max: 45 })
    expect(getExtent(curve([0, 20, 5]))).toStrictEqual({ min: 0, max: 20 })
    expect(getExtent(curve([]))).toStrictEqual({ min: 0, max: 0 })
  })

  // Whatever the run holds, the height read as nothing is the foot line.
  it('draws nothing on the foot under every run', () => {
    for (const run of [
      [0, 0, 0],
      [0, 20, 5],
      [40, 45, 41],
    ]) {
      const one = curve(run)
      const positions = createPositions(one, getExtent(one))
      for (const [index, value] of run.entries()) {
        if (value === 0) expect(positions[index]?.y).toBe(PLOT_BOTTOM)
        expect(positions[index]?.y).toBeLessThanOrEqual(PLOT_BOTTOM)
      }
    }
  })

  it('is one line through every place', () => {
    const positions = createPositions(curve([0, 0.5, 1]), { min: 0, max: 1 })
    expect(createLinePath(positions).startsWith('M')).toBe(true)
    expect(createLinePath(positions).split('L')).toHaveLength(3)
  })

  it('is nothing at all where the curve holds no place', () => {
    expect(createLinePath([])).toBe('')
  })
})

describe('the stretch a budget does not get through', () => {
  it('is a line of its own over the places the budget falls short at', () => {
    const one = curve([0, 0.4, 0.7, 1], [true, false, false, true])
    expect(getShortPath(one, createPositions(one, getExtent(one))).startsWith('M')).toBe(true)
  })

  it('is nothing where the budget gets through all of it', () => {
    const one = curve([0, 0.4, 1])
    expect(getShortPath(one, createPositions(one, getExtent(one)))).toBe('')
  })

  it('is nothing where it falls short at one place alone, which draws no line', () => {
    const one = curve([0, 0.4, 1], [true, false, true])
    expect(getShortPath(one, createPositions(one, getExtent(one)))).toBe('')
  })
})

describe('the place a pointer remains over', () => {
  it('is the first at the left edge of the room and the last at the right', () => {
    expect(getPlaceAtX(PLOT_LEFT, 25)).toBe(0)
    expect(getPlaceAtX(PLOT_RIGHT, 25)).toBe(24)
  })

  it('is the nearer end for a pointer outside the room the line is drawn in', () => {
    expect(getPlaceAtX(0, 25)).toBe(0)
    expect(getPlaceAtX(PLOT_WIDTH, 25)).toBe(24)
  })

  it('reads back the place a point was drawn at', () => {
    const one = curve([0, 0.25, 0.5, 0.75, 1])
    const positions = createPositions(one, getExtent(one))
    positions.forEach((point, place) => expect(getPlaceAtX(point.x, positions.length)).toBe(place))
  })

  it('is the first place of a grid holding one', () => {
    expect(getPlaceAtX(PLOT_WIDTH / 2, 1)).toBe(0)
  })

  it('is the first place of a grid holding none', () => {
    expect(getPlaceAtX(PLOT_WIDTH / 2, 0)).toBe(0)
  })
})

describe('where a keystroke takes the knob', () => {
  it('is one place either way, and stops at either end', () => {
    expect(getKeyPlace('ArrowRight', 2, 5)).toBe(3)
    expect(getKeyPlace('ArrowLeft', 2, 5)).toBe(1)
    expect(getKeyPlace('ArrowLeft', 0, 5)).toBe(0)
    expect(getKeyPlace('ArrowRight', 4, 5)).toBe(4)
  })

  it('is the same either way for the two axes, so a knob answers both', () => {
    expect(getKeyPlace('ArrowDown', 2, 5)).toBe(getKeyPlace('ArrowLeft', 2, 5))
    expect(getKeyPlace('ArrowUp', 2, 5)).toBe(getKeyPlace('ArrowRight', 2, 5))
  })

  it('is either end of the range', () => {
    expect(getKeyPlace('Home', 2, 5)).toBe(0)
    expect(getKeyPlace('End', 2, 5)).toBe(4)
  })

  it('is nothing for a keystroke of somebody else’s', () => {
    expect(getKeyPlace('a', 2, 5)).toBeNull()
    expect(getKeyPlace('Enter', 2, 5)).toBeNull()
  })
})

describe('the extent a backlog is drawn against', () => {
  it('relies on nothing and reaches the most the pace ever remains at', () => {
    expect(getBacklogExtent([3, 7, 2])).toStrictEqual({ min: 0, max: 7 })
  })

  it('is nothing at all for a pace holding no days', () => {
    expect(getBacklogExtent([])).toStrictEqual({ min: 0, max: 0 })
  })
})

describe('a series drawn against an extent that does not begin at nothing', () => {
  it('measures each value up from the least of the extent', () => {
    const series = createSeries([10, 15, 20], { min: 10, max: 20 })
    expect(series.map((one) => one.y)).toStrictEqual([
      PLOT_BOTTOM,
      PLOT_BOTTOM - (PLOT_BOTTOM - PLOT_TOP) / 2,
      PLOT_TOP,
    ])
  })
})

describe('the line through places', () => {
  it('writes each place as its two figures, rounded to two places, after M and then L', () => {
    expect(
      createLinePath([
        { x: 1.234, y: 5.678 },
        { x: 10, y: 20 },
        { x: 30, y: 40.5 },
      ]),
    ).toBe('M1.23 5.68 L10 20 L30 40.5')
  })
})

describe('the stretch a budget does not get through, as drawn', () => {
  const axisPoints = (count: number) =>
    Array.from({ length: count }, (_, index) => ({ x: index, y: index }))

  it('is one line through the places in a stretch', () => {
    const one = curve([0, 0, 0, 0], [true, false, false, true])
    expect(getShortPath(one, axisPoints(4))).toBe('M1 1 L2 2')
  })

  it('is one line to a stretch, set apart by a space', () => {
    const one = curve([0, 0, 0, 0, 0], [false, false, true, false, false])
    expect(getShortPath(one, axisPoints(5))).toBe('M0 0 L1 1 M3 3 L4 4')
  })

  it('draws no line for a single place at the start', () => {
    expect(getShortPath(curve([0, 0, 0], [false, true, true]), axisPoints(3))).toBe('')
  })

  it('draws no line for a single place at the end', () => {
    expect(getShortPath(curve([0, 0], [true, false]), axisPoints(2))).toBe('')
  })

  it('draws the line of a stretch that runs to the end', () => {
    expect(getShortPath(curve([0, 0, 0], [true, false, false]), axisPoints(3))).toBe('M1 1 L2 2')
  })

  it('leaves out the places of the curve that have no position', () => {
    expect(getShortPath(curve([0, 0, 0], [false, false, false]), axisPoints(1))).toBe('')
  })
})

describe('what remains overdue at one place', () => {
  const backlogs = (goal: 'date' | 'minutes', grid: readonly number[]): Curve => {
    const base = curve([0, 0, 0])
    return {
      ...base,
      goal,
      grid,
      at: base.at.map((one) => ({ ...one, backlog: [4, 5, 6, 7] })),
    }
  }

  it('is the whole run for a goal that is not a date', () => {
    expect(getBacklogRun(backlogs('minutes', [0, 1, 2]), 1)).toStrictEqual([4, 5, 6, 7])
  })

  it('is cut at the day a goal of a date names', () => {
    expect(getBacklogRun(backlogs('date', [0, 2, 9]), 1)).toStrictEqual([4, 5])
    expect(getBacklogRun(backlogs('date', [0, 2, 9]), 2)).toStrictEqual([4, 5, 6, 7])
  })

  it('is nothing at a goal of a date that names the first day', () => {
    expect(getBacklogRun(backlogs('date', [0, 2, 9]), 0)).toStrictEqual([])
  })

  it('is nothing at a place the curve does not have', () => {
    expect(getBacklogRun(backlogs('minutes', [0, 1, 2]), 7)).toStrictEqual([])
  })
})
