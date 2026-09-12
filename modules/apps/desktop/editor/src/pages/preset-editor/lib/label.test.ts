/**
 * The words set on the one control's picture, and the room each takes.
 *
 * A word is placed against a line or over a mark, and what it comes to is a
 * box. Two boxes that touch are two words that would print over each other,
 * and the picture drops one of them.
 */
import { describe, expect, it } from 'vitest'

import {
  AXIS_HIGH,
  AXIS_WIDE,
  calloutOf,
  CALLOUT_GAP,
  CALLOUT_HIGH,
  CALLOUT_WIDE,
  getAxisNumber,
  getAxisNumberBox,
  getNameBox,
  heightsOf,
  LABEL,
  labelsOf,
  LIFT,
  positionLabel,
  readingAt,
  type Box,
  type Mark,
} from './label'
import { BACKLOG_PLOT, FOOT, HIGH, LEFT, RIGHT, TOP, WIDE, type Position } from './plot'

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
    const room = getNameBox({ x: WIDE / 2, y: 100 })

    expect(room).toStrictEqual({
      x: WIDE / 2 - LABEL,
      y: 100 - LIFT - AXIS_HIGH,
      wide: LABEL * 2,
      high: AXIS_HIGH,
    })
  })

  it('is drawn from the mark at the near end, and back from it at the far one', () => {
    expect(getNameBox({ x: LEFT, y: 100 }).x).toBe(LEFT)
    expect(getNameBox({ x: RIGHT, y: 100 }).x).toBe(RIGHT - LABEL * 2)
  })
})

describe('where a number against a line is set', () => {
  it('stands at the left of the picture, at the height it is read off', () => {
    const at = getAxisNumber(FOOT, '0')

    expect(at.insetInlineStart).toBe(`${(LEFT / WIDE) * 100}%`)
    expect(at.insetBlockStart).toBe(`${(FOOT / HIGH) * 100}%`)
    expect(at.translate).toBe('0 0')
  })

  it('is read against the room it is given, which the extent under the picture has its own of', () => {
    expect(getAxisNumber(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high).insetBlockStart).toBe(
      `${(BACKLOG_PLOT.foot / BACKLOG_PLOT.high) * 100}%`,
    )
  })
})

describe('the room a number against a line takes', () => {
  it('hangs below the height where it is set on it', () => {
    expect(getAxisNumberBox(FOOT, '0')).toStrictEqual({
      x: LEFT,
      y: FOOT,
      wide: AXIS_WIDE,
      high: AXIS_HIGH,
    })
  })

  it('stands above the height where it is lifted off it', () => {
    expect(getAxisNumberBox(TOP, '-100%').y).toBe(TOP - AXIS_HIGH)
  })
})

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
        box: getNameBox({ x: WIDE / 2, y: 100 }),
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
const formatValue = (value: number): string => String(value)

/** A curve drawn well clear of the left edge, where no number is read. */
const clear: readonly Position[] = [
  { x: 300, y: TOP },
  { x: RIGHT, y: FOOT },
]

describe('the numbers read off the picture’s edges', () => {
  it('is the most over the top and the least on the foot', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [], null, formatValue)

    expect(numbers.map((one) => one.text)).toStrictEqual(['10', '0'])
    expect(numbers.map((one) => one.box)).toStrictEqual([
      getAxisNumberBox(TOP, '-100%'),
      getAxisNumberBox(FOOT, '0'),
    ])
  })

  // An extent of no width has one number and nothing else to read.
  it('is one number on the foot for an extent of no width', () => {
    const numbers = heightsOf({ least: 4, most: 4 }, clear, [], null, formatValue)

    expect(numbers).toStrictEqual([
      { at: getAxisNumber(FOOT, '0'), box: getAxisNumberBox(FOOT, '0'), text: '4' },
    ])
  })

  it('drops the number the curve itself stands on', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, [{ x: LEFT, y: FOOT }], [], null, formatValue)

    expect(numbers.map((one) => one.text)).toStrictEqual(['10'])
  })

  it('drops the number a mark stands on', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [{ x: LEFT, y: TOP }], null, formatValue)

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('drops the number the bubble over the knob stands on', () => {
    const over = box(LEFT, 0, CALLOUT_WIDE, CALLOUT_HIGH)
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [], over, formatValue)

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('keeps every number it draws inside the picture', () => {
    const numbers = heightsOf({ least: 0, most: 10 }, clear, [], null, formatValue)

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
