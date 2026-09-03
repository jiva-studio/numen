/**
 * Where the one control's curve is drawn, and where a pointer over it stands.
 *
 * The picture is the control, so the same arithmetic is asked both ways: a
 * place put on the line, and a place read back off a pointer.
 */
import { describe, expect, it } from 'vitest'

import { NOWHERE, type Curve, type Point } from './core'
import {
  against,
  againstBox,
  AXIS_HIGH,
  AXIS_WIDE,
  BAND,
  bandOf,
  FOOT,
  HIGH,
  LABEL,
  LEFT,
  LIFT,
  lineOf,
  naming,
  namingBox,
  placeUnder,
  RIGHT,
  shortOf,
  spotsOf,
  TOP,
  walked,
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
  })

  it('keeps a curve past either end of its band inside the picture', () => {
    const spots = spotsOf(curve([-1, 0.5, 2]), { least: 0, most: 1 })
    expect(spots[0]?.y).toBe(FOOT)
    expect(spots[2]?.y).toBe(TOP)
  })

  // A count does not go below nothing, so nothing is the foot of the picture
  // and a run of it lies along that foot.
  it('lies along the floor where the band has no width', () => {
    const spots = spotsOf(curve([0, 0, 0]), { least: 0, most: 0 })
    expect(spots.every((one) => one.y === FOOT)).toBe(true)
  })

  it('stands a band on nothing, whatever the run it holds comes to', () => {
    expect(bandOf(curve([0, 0, 0]))).toStrictEqual({ least: 0, most: 0 })
    expect(bandOf(curve([40, 45, 41]))).toStrictEqual({ least: 0, most: 45 })
    expect(bandOf(curve([0, 20, 5]))).toStrictEqual({ least: 0, most: 20 })
    expect(bandOf(curve([]))).toStrictEqual({ least: 0, most: 0 })
  })

  // Whatever the run holds, the height read as nothing is the foot line.
  it('draws nothing on the foot under every run', () => {
    for (const run of [[0, 0, 0], [0, 20, 5], [40, 45, 41]]) {
      const one = curve(run)
      const spots = spotsOf(one, bandOf(one))
      for (const [at, value] of run.entries()) {
        if (value === 0) expect(spots[at]?.y).toBe(FOOT)
        expect(spots[at]?.y).toBeLessThanOrEqual(FOOT)
      }
    }
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

describe('where a name over a mark is set', () => {
  it('stands over the mark, and above it by the lift', () => {
    const at = naming({ x: WIDE / 2, y: 100 })

    expect(at.insetInlineStart).toBe('50%')
    expect(at.insetBlockStart).toBe(`${((100 - LIFT) / HIGH) * 100}%`)
    expect(at.translate).toBe('-50% -100%')
  })

  it('is pulled back inside the picture at either end', () => {
    expect(naming({ x: LEFT, y: 100 }).translate).toBe('0 -100%')
    expect(naming({ x: RIGHT, y: 100 }).translate).toBe('-100% -100%')
  })

  it('is held inside the top of the picture, so no word is set over the edge', () => {
    expect(naming({ x: WIDE / 2, y: TOP }).insetBlockStart).toBe(`${(TOP / HIGH) * 100}%`)
  })
})

describe('the room a name over a mark takes', () => {
  it('stands where the name does, and is as wide as a label either side', () => {
    const box = namingBox({ x: WIDE / 2, y: 100 })

    expect(box).toStrictEqual({
      x: WIDE / 2 - LABEL,
      y: 100 - LIFT - AXIS_HIGH,
      wide: LABEL * 2,
      high: AXIS_HIGH,
    })
  })

  it('is drawn from the mark at the near end, and back from it at the far one', () => {
    expect(namingBox({ x: LEFT, y: 100 }).x).toBe(LEFT)
    expect(namingBox({ x: RIGHT, y: 100 }).x).toBe(RIGHT - LABEL * 2)
  })
})

describe('where a number against a line is set', () => {
  it('stands at the left of the picture, at the height it is read off', () => {
    const at = against(FOOT, '0')

    expect(at.insetInlineStart).toBe(`${(LEFT / WIDE) * 100}%`)
    expect(at.insetBlockStart).toBe(`${(FOOT / HIGH) * 100}%`)
    expect(at.translate).toBe('0 0')
  })

  it('is read against the room it is given, which the band under the picture has its own of', () => {
    expect(against(BAND.foot, '0', BAND.high).insetBlockStart).toBe(
      `${(BAND.foot / BAND.high) * 100}%`,
    )
  })
})

describe('the room a number against a line takes', () => {
  it('hangs below the height where it is set on it', () => {
    expect(againstBox(FOOT, '0')).toStrictEqual({
      x: LEFT,
      y: FOOT,
      wide: AXIS_WIDE,
      high: AXIS_HIGH,
    })
  })

  it('stands above the height where it is lifted off it', () => {
    expect(againstBox(TOP, '-100%').y).toBe(TOP - AXIS_HIGH)
  })
})

describe('where a keystroke takes the knob', () => {
  it('is one place either way, and stops at either end', () => {
    expect(walked('ArrowRight', 2, 5)).toBe(3)
    expect(walked('ArrowLeft', 2, 5)).toBe(1)
    expect(walked('ArrowLeft', 0, 5)).toBe(0)
    expect(walked('ArrowRight', 4, 5)).toBe(4)
  })

  it('is the same either way for the two axes, so a knob answers both', () => {
    expect(walked('ArrowDown', 2, 5)).toBe(walked('ArrowLeft', 2, 5))
    expect(walked('ArrowUp', 2, 5)).toBe(walked('ArrowRight', 2, 5))
  })

  it('is either end of the range', () => {
    expect(walked('Home', 2, 5)).toBe(0)
    expect(walked('End', 2, 5)).toBe(4)
  })

  it('is nothing for a keystroke of somebody else’s', () => {
    expect(walked('a', 2, 5)).toBeNull()
    expect(walked('Enter', 2, 5)).toBeNull()
  })
})
