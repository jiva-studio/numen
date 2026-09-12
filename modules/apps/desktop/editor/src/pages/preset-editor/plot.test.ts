/**
 * Where the one control's curve is drawn, and where a pointer over it stands.
 *
 * The picture is the control, so the same arithmetic is asked both ways: a
 * place put on the line, and a place read back off a pointer.
 */
import { describe, expect, it } from 'vitest'

import { NOWHERE, type Curve, type Point } from './types'
import {
  against,
  againstBox,
  AXIS_HIGH,
  AXIS_WIDE,
  BACKLOG_PLOT,
  extentOf,
  FOOT,
  heightsOf,
  HIGH,
  LABEL,
  labelsOf,
  LEFT,
  LIFT,
  lineOf,
  positionLabel,
  namingBox,
  calloutOf,
  CALLOUT_GAP,
  CALLOUT_HIGH,
  CALLOUT_WIDE,
  placeUnder,
  positionsOf,
  readingAt,
  RIGHT,
  shortOf,
  TOP,
  walkGrid,
  WIDE,
  xOf,
  yOfGridline,
  type Box,
  type Mark,
  type Position,
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

describe('where a name over a mark is set', () => {
  it('stands over the mark, and above it by the lift', () => {
    const at = positionLabel({ x: WIDE / 2, y: 100 })

    expect(at.insetInlineStart).toBe('50%')
    expect(at.insetBlockStart).toBe(`${((100 - LIFT) / HIGH) * 100}%`)
    expect(at.translate).toBe('-50% -100%')
  })

  it('is pulled back inside the picture at either end', () => {
    expect(positionLabel({ x: LEFT, y: 100 }).translate).toBe('0 -100%')
    expect(positionLabel({ x: RIGHT, y: 100 }).translate).toBe('-100% -100%')
  })

  it('is held inside the top of the picture, so no word is set over the edge', () => {
    expect(positionLabel({ x: WIDE / 2, y: TOP }).insetBlockStart).toBe(`${(TOP / HIGH) * 100}%`)
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

  it('is read against the room it is given, which the extent under the picture has its own of', () => {
    expect(against(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high).insetBlockStart).toBe(
      `${(BACKLOG_PLOT.foot / BACKLOG_PLOT.high) * 100}%`,
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

const mark = (key: string, x: number, y: number, text = key): Mark => ({
  key,
  at: { x, y },
  text,
})

/** A room somewhere in the picture, to stand a word against. */
const box = (x: number, y: number, wide: number, high: number): Box => ({ x, y, wide, high })

/** Whether a room is wholly inside the picture, which is what the viewBox holds. */
const inside = (one: Box): boolean =>
  one.x >= 0 && one.y >= 0 && one.x + one.wide <= WIDE && one.y + one.high <= HIGH

describe('the bubble over the knob', () => {
  it('hangs above the knob, clear of it by the gap', () => {
    const callout = calloutOf({ x: WIDE / 2, y: 120 })

    expect(callout.under).toBe(false)
    expect(callout.box).toStrictEqual(
      box(WIDE / 2 - CALLOUT_WIDE / 2, 120 - CALLOUT_GAP - CALLOUT_HIGH, CALLOUT_WIDE, CALLOUT_HIGH),
    )
  })

  // Above would take it off the top, so it turns over and hangs under instead.
  it('turns under the knob where above would take it off the top', () => {
    const callout = calloutOf({ x: WIDE / 2, y: TOP })

    expect(callout.under).toBe(true)
    expect(callout.box.y).toBe(TOP + CALLOUT_GAP)
    expect(callout.at.translate).toBe('-50% 0')
  })

  it('turns over at the exact height it no longer fits above', () => {
    expect(calloutOf({ x: WIDE / 2, y: TOP + CALLOUT_GAP + CALLOUT_HIGH }).under).toBe(false)
    expect(calloutOf({ x: WIDE / 2, y: TOP + CALLOUT_GAP + CALLOUT_HIGH - 1 }).under).toBe(true)
  })

  it.each([
    { where: 'the left edge', knob: { x: LEFT, y: 120 }, back: '0%' },
    { where: 'the middle', knob: { x: WIDE / 2, y: 120 }, back: '-50%' },
    { where: 'the right edge', knob: { x: RIGHT, y: 120 }, back: '-100%' },
  ])('is pulled back inside the picture at $where', ({ knob, back }) => {
    const callout = calloutOf(knob)

    expect(callout.at.translate).toBe(`${back} -100%`)
    expect(inside(callout.box)).toBe(true)
  })

  it('anchors its tail on the knob, wherever the bubble was pulled to', () => {
    const callout = calloutOf({ x: LEFT, y: 120 })

    expect(callout.tail.insetInlineStart).toBe(callout.at.insetInlineStart)
    expect(callout.tail.insetBlockStart).toBe(callout.at.insetBlockStart)
  })
})

describe('the names of the marks that fit', () => {
  it('says nothing where there are no marks', () => {
    expect(labelsOf([], null)).toStrictEqual([])
  })

  it('sets the one name a single mark carries', () => {
    expect(labelsOf([mark('suggested', WIDE / 2, 100)], null)).toStrictEqual([
      {
        key: 'suggested',
        text: 'suggested',
        at: positionLabel({ x: WIDE / 2, y: 100 }),
        box: namingBox({ x: WIDE / 2, y: 100 }),
      },
    ])
  })

  it('leaves off a mark carrying no name at all', () => {
    expect(labelsOf([mark('knob', WIDE / 2, 100, '')], null)).toStrictEqual([])
  })

  // A name that will not fit is dropped, not moved: nothing here pushes two
  // names apart, so the second of a touching pair is simply not drawn.
  it.each([
    { gap: LABEL * 2 - 1, kept: ['first'] },
    { gap: LABEL * 2, kept: ['first', 'second'] },
    { gap: LABEL * 2 + 1, kept: ['first', 'second'] },
  ])('keeps $kept.length of two names $gap apart at the same height', ({ gap, kept }) => {
    const names = labelsOf([mark('first', 200, 100), mark('second', 200 + gap, 100)], null)

    expect(names.map((one) => one.key)).toStrictEqual(kept)
  })

  it('keeps two names at the same place where their heights stand clear', () => {
    const names = labelsOf([mark('first', 200, 60), mark('second', 200, 60 + LIFT + AXIS_HIGH)], null)

    expect(names.map((one) => one.key)).toStrictEqual(['first', 'second'])
  })

  it('leaves off a name the bubble over the knob stands on', () => {
    const over = calloutOf({ x: 280, y: TOP }).box

    expect(labelsOf([mark('suggested', 280, 100)], over)).toStrictEqual([])
    expect(labelsOf([mark('suggested', 280, 100)], null)).toHaveLength(1)
  })

  it.each([
    { where: 'the left edge', at: { x: LEFT, y: 100 } },
    { where: 'the right edge', at: { x: RIGHT, y: 100 } },
    { where: 'the top', at: { x: WIDE / 2, y: TOP } },
    { where: 'the foot', at: { x: WIDE / 2, y: FOOT } },
  ])('keeps the room a name takes at $where inside the picture', ({ at }) => {
    const names = labelsOf([mark('suggested', at.x, at.y)], null)

    expect(names).toHaveLength(1)
    expect(inside(names[0]!.box)).toBe(true)
  })
})

/** A number said as itself, so a test reads the height and not the wording. */
const said = (value: number): string => String(value)

/** A curve drawn well clear of the left edge, where no number is read. */
const clear: readonly Position[] = [
  { x: 300, y: TOP },
  { x: RIGHT, y: FOOT },
]

describe('the numbers read off the picture’s edges', () => {
  it('is the most over the top and the least on the foot', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [], null, said)

    expect(numbers.map((one) => one.text)).toStrictEqual(['10', '0'])
    expect(numbers.map((one) => one.box)).toStrictEqual([
      againstBox(TOP, '-100%'),
      againstBox(FOOT, '0'),
    ])
  })

  // An extent of no width has one number and nothing else to read.
  it('is one number on the foot for an extent of no width', () => {
    const numbers = heightsOf({ least: 4, most: 4 }, clear, [], null, said)

    expect(numbers).toStrictEqual([
      { at: against(FOOT, '0'), box: againstBox(FOOT, '0'), text: '4' },
    ])
  })

  it('drops the number the curve itself stands on', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, [{ x: LEFT, y: FOOT }], [], null, said)

    expect(numbers.map((one) => one.text)).toStrictEqual(['10'])
  })

  it('drops the number a mark stands on', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [{ x: LEFT, y: TOP }], null, said)

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('drops the number the bubble over the knob stands on', () => {
    const over = box(LEFT, 0, CALLOUT_WIDE, CALLOUT_HIGH)
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [], over, said)

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('keeps every number it draws inside the picture', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [], null, said)

    expect(numbers.every((one) => inside(one.box))).toBe(true)
  })
})

describe('where the knob’s own value is set', () => {
  it('is nowhere at all while the knob stands nowhere', () => {
    expect(readingAt(null)).toStrictEqual({})
  })

  it.each([
    { where: 'the left edge', x: LEFT, back: '0' },
    { where: 'the middle', x: WIDE / 2, back: '-50%' },
    { where: 'the right edge', x: RIGHT, back: '-100%' },
  ])('is pulled back inside the picture at $where', ({ x, back }) => {
    const at = readingAt({ x, y: 100 })

    expect(at.insetInlineStart).toBe(`${(x / WIDE) * 100}%`)
    expect(at.translate).toBe(`${back} 0`)
  })
})
