/**
 * The words set on the one control's picture, and the room each takes.
 *
 * A word is placed against a line or over a mark, and what it evaluates to is a
 * box. Two boxes that touch are two words that would print over each other,
 * and the picture drops one of them.
 */
import { describe, expect, it } from 'vitest'

import {
  AXIS_LABEL_HEIGHT,
  AXIS_LABEL_WIDTH,
  createCallout,
  CALLOUT_GAP,
  CALLOUT_HEIGHT,
  CALLOUT_WIDTH,
  getAxisNumber,
  getAxisNumberBox,
  isAxisLabelClear,
  getNameBox,
  areBoxesSeparate,
  createAxisLabels,
  LABEL,
  createLabels,
  AXIS_LABEL_LIFT,
  getMarkNameStyle,
  getKnobValueStyle,
  type Box,
  type Mark,
} from './label'
import {
  BACKLOG_PLOT,
  PLOT_BOTTOM,
  PLOT_HEIGHT,
  PLOT_LEFT,
  PLOT_RIGHT,
  PLOT_TOP,
  PLOT_WIDTH,
  type Position,
} from './plot'

const mark = (key: string, x: number, y: number, text = key): Mark => ({
  key,
  at: { x, y },
  text,
})

/** A room somewhere in the picture, to stand a word against. */
const box = (x: number, y: number, width: number, height: number): Box => ({ x, y, wide: width, high: height })

/** Whether a room is wholly inside the picture, which is what the viewBox holds. */
const isInside = (one: Box): boolean =>
  one.x >= 0 && one.y >= 0 && one.x + one.wide <= PLOT_WIDTH && one.y + one.high <= PLOT_HEIGHT

describe('where a name over a mark is set', () => {
  it('remains over the mark, and above it by the lift', () => {
    const style = getMarkNameStyle({ x: PLOT_WIDTH / 2, y: 100 })

    expect(style.insetInlineStart).toBe('50%')
    expect(style.insetBlockStart).toBe(`${((100 - AXIS_LABEL_LIFT) / PLOT_HEIGHT) * 100}%`)
    expect(style.translate).toBe('-50% -100%')
  })

  it('is pulled back inside the picture at either end', () => {
    expect(getMarkNameStyle({ x: PLOT_LEFT, y: 100 }).translate).toBe('0 -100%')
    expect(getMarkNameStyle({ x: PLOT_RIGHT, y: 100 }).translate).toBe('-100% -100%')
  })

  it('is retained inside the top of the picture, so no word is set over the edge', () => {
    expect(getMarkNameStyle({ x: PLOT_WIDTH / 2, y: PLOT_TOP }).insetBlockStart).toBe(
      `${(PLOT_TOP / PLOT_HEIGHT) * 100}%`,
    )
  })
})

describe('the room a name over a mark takes', () => {
  it('remains where the name does, and is as wide as a label either side', () => {
    const room = getNameBox({ x: PLOT_WIDTH / 2, y: 100 })

    expect(room).toStrictEqual({
      x: PLOT_WIDTH / 2 - LABEL,
      y: 100 - AXIS_LABEL_LIFT - AXIS_LABEL_HEIGHT,
      wide: LABEL * 2,
      high: AXIS_LABEL_HEIGHT,
    })
  })

  it('is drawn from the mark at the near end, and back from it at the far one', () => {
    expect(getNameBox({ x: PLOT_LEFT, y: 100 }).x).toBe(PLOT_LEFT)
    expect(getNameBox({ x: PLOT_RIGHT, y: 100 }).x).toBe(PLOT_RIGHT - LABEL * 2)
  })
})

describe('where a number against a line is set', () => {
  it('remains at the left of the picture, at the height it is read off', () => {
    const style = getAxisNumber(PLOT_BOTTOM, '0')

    expect(style.insetInlineStart).toBe(`${(PLOT_LEFT / PLOT_WIDTH) * 100}%`)
    expect(style.insetBlockStart).toBe(`${(PLOT_BOTTOM / PLOT_HEIGHT) * 100}%`)
    expect(style.translate).toBe('0 0')
  })

  it('is read against the room it is given, which the extent under the picture has its own of', () => {
    expect(getAxisNumber(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high).insetBlockStart).toBe(
      `${(BACKLOG_PLOT.foot / BACKLOG_PLOT.high) * 100}%`,
    )
  })
})

describe('the room a number against a line takes', () => {
  it('hangs below the height where it is set on it', () => {
    expect(getAxisNumberBox(PLOT_BOTTOM, '0')).toStrictEqual({
      x: PLOT_LEFT,
      y: PLOT_BOTTOM,
      wide: AXIS_LABEL_WIDTH,
      high: AXIS_LABEL_HEIGHT,
    })
  })

  it('remains above the height where it is lifted off it', () => {
    expect(getAxisNumberBox(PLOT_TOP, '-100%').y).toBe(PLOT_TOP - AXIS_LABEL_HEIGHT)
  })
})

describe('the bubble over the knob', () => {
  it('hangs above the knob, clear of it by the gap', () => {
    const callout = createCallout({ x: PLOT_WIDTH / 2, y: 120 })

    expect(callout.isUnder).toBe(false)
    expect(callout.box).toStrictEqual(
      box(
        PLOT_WIDTH / 2 - CALLOUT_WIDTH / 2,
        120 - CALLOUT_GAP - CALLOUT_HEIGHT,
        CALLOUT_WIDTH,
        CALLOUT_HEIGHT,
      ),
    )
  })

  // Above would take it off the top, so it turns over and hangs under instead.
  it('turns under the knob where above would take it off the top', () => {
    const callout = createCallout({ x: PLOT_WIDTH / 2, y: PLOT_TOP })

    expect(callout.isUnder).toBe(true)
    expect(callout.box.y).toBe(PLOT_TOP + CALLOUT_GAP)
    expect(callout.at.translate).toBe('-50% 0')
  })

  it('turns over at the exact height it no longer fits above', () => {
    expect(
      createCallout({ x: PLOT_WIDTH / 2, y: PLOT_TOP + CALLOUT_GAP + CALLOUT_HEIGHT }).isUnder,
    ).toBe(false)
    expect(
      createCallout({ x: PLOT_WIDTH / 2, y: PLOT_TOP + CALLOUT_GAP + CALLOUT_HEIGHT - 1 }).isUnder,
    ).toBe(true)
  })

  it.each([
    { where: 'the left edge', knob: { x: PLOT_LEFT, y: 120 }, back: '0%' },
    { where: 'the middle', knob: { x: PLOT_WIDTH / 2, y: 120 }, back: '-50%' },
    { where: 'the right edge', knob: { x: PLOT_RIGHT, y: 120 }, back: '-100%' },
  ])('is pulled back inside the picture at $where', ({ knob, back }) => {
    const callout = createCallout(knob)

    expect(callout.at.translate).toBe(`${back} -100%`)
    expect(isInside(callout.box)).toBe(true)
  })

  it('anchors its tail on the knob, wherever the bubble was pulled to', () => {
    const callout = createCallout({ x: PLOT_LEFT, y: 120 })

    expect(callout.tail.insetInlineStart).toBe(callout.at.insetInlineStart)
    expect(callout.tail.insetBlockStart).toBe(callout.at.insetBlockStart)
  })
})

describe('the names of the marks that fit', () => {
  it('says nothing where there are no marks', () => {
    expect(createLabels([], null)).toStrictEqual([])
  })

  it('sets the one name a single mark carries', () => {
    expect(createLabels([mark('suggested', PLOT_WIDTH / 2, 100)], null)).toStrictEqual([
      {
        key: 'suggested',
        text: 'suggested',
        at: getMarkNameStyle({ x: PLOT_WIDTH / 2, y: 100 }),
        box: getNameBox({ x: PLOT_WIDTH / 2, y: 100 }),
      },
    ])
  })

  it('leaves off a mark carrying no name at all', () => {
    expect(createLabels([mark('knob', PLOT_WIDTH / 2, 100, '')], null)).toStrictEqual([])
  })

  // A name that will not fit is dropped, not moved: nothing here pushes two
  // names apart, so the second of a touching pair is simply not drawn.
  it.each([
    { gap: LABEL * 2 - 1, expected: ['first'] },
    { gap: LABEL * 2, expected: ['first', 'second'] },
    { gap: LABEL * 2 + 1, expected: ['first', 'second'] },
  ])('keeps $kept.length of two names $gap apart at the same height', ({ gap, expected }) => {
    const names = createLabels([mark('first', 200, 100), mark('second', 200 + gap, 100)], null)

    expect(names.map((one) => one.key)).toStrictEqual(expected)
  })

  it('keeps two names at the same place where their heights stand clear', () => {
    const names = createLabels(
      [mark('first', 200, 60), mark('second', 200, 60 + AXIS_LABEL_LIFT + AXIS_LABEL_HEIGHT)],
      null,
    )

    expect(names.map((one) => one.key)).toStrictEqual(['first', 'second'])
  })

  it('leaves off a name the bubble over the knob relies on', () => {
    const calloutBox = createCallout({ x: 280, y: PLOT_TOP }).box

    expect(createLabels([mark('suggested', 280, 100)], calloutBox)).toStrictEqual([])
    expect(createLabels([mark('suggested', 280, 100)], null)).toHaveLength(1)
  })

  it.each([
    { where: 'the left edge', at: { x: PLOT_LEFT, y: 100 } },
    { where: 'the right edge', at: { x: PLOT_RIGHT, y: 100 } },
    { where: 'the top', at: { x: PLOT_WIDTH / 2, y: PLOT_TOP } },
    { where: 'the foot', at: { x: PLOT_WIDTH / 2, y: PLOT_BOTTOM } },
  ])('keeps the room a name takes at $where inside the picture', ({ at: pos }) => {
    const names = createLabels([mark('suggested', pos.x, pos.y)], null)

    expect(names).toHaveLength(1)
    expect(isInside(names[0]!.box)).toBe(true)
  })
})

/** A number said as itself, so a test reads the height and not the wording. */
const formatValue = (value: number): string => String(value)

/** A curve drawn well clear of the left edge, where no number is read. */
const clear: readonly Position[] = [
  { x: 300, y: PLOT_TOP },
  { x: PLOT_RIGHT, y: PLOT_BOTTOM },
]

describe('the numbers read off the picture’s edges', () => {
  it('is the most over the top and the least on the foot', () => {
    const numbers = createAxisLabels(
      { least: 0, most: 10 },
      { places: clear, marks: [] },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['10', '0'])
    expect(numbers.map((one) => one.box)).toStrictEqual([
      getAxisNumberBox(PLOT_TOP, '-100%'),
      getAxisNumberBox(PLOT_BOTTOM, '0'),
    ])
  })

  // An extent of no width has one number and nothing else to read.
  it('is one number on the foot for an extent of no width', () => {
    const numbers = createAxisLabels(
      { least: 4, most: 4 },
      { places: clear, marks: [] },
      formatValue,
    )

    expect(numbers).toStrictEqual([
      { at: getAxisNumber(PLOT_BOTTOM, '0'), box: getAxisNumberBox(PLOT_BOTTOM, '0'), text: '4' },
    ])
  })

  it('drops the number the curve itself relies on', () => {
    const numbers = createAxisLabels(
      { least: 0, most: 10 },
      { places: [{ x: PLOT_LEFT, y: PLOT_BOTTOM }], marks: [] },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['10'])
  })

  it('drops the number a mark relies on', () => {
    const numbers = createAxisLabels(
      { least: 0, most: 10 },
      { places: clear, marks: [{ x: PLOT_LEFT, y: PLOT_TOP }] },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('drops the number the bubble over the knob relies on', () => {
    const calloutBox = box(PLOT_LEFT, 0, CALLOUT_WIDTH, CALLOUT_HEIGHT)
    const numbers = createAxisLabels(
      { least: 0, most: 10 },
      { places: clear, marks: [], calloutBox },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('keeps every number it draws inside the picture', () => {
    const numbers = createAxisLabels(
      { least: 0, most: 10 },
      { places: clear, marks: [] },
      formatValue,
    )

    expect(numbers.every((one) => isInside(one.box))).toBe(true)
  })
})

describe('where the knob’s own value is set', () => {
  it('is nowhere at all while the knob remains nowhere', () => {
    expect(getKnobValueStyle(null)).toStrictEqual({})
  })

  it.each([
    { where: 'the left edge', x: PLOT_LEFT, back: '0' },
    { where: 'the middle', x: PLOT_WIDTH / 2, back: '-50%' },
    { where: 'the right edge', x: PLOT_RIGHT, back: '-100%' },
  ])('is pulled back inside the picture at $where', ({ x, back }) => {
    const style = getKnobValueStyle({ x, y: 100 })

    expect(style.insetInlineStart).toBe(`${(x / PLOT_WIDTH) * 100}%`)
    expect(style.translate).toBe(`${back} 0`)
  })
})

describe('two words that stand clear of each other', () => {
  const one = box(100, 100, 40, 20)

  it.each([
    { where: 'to the right', other: box(140, 100, 40, 20) },
    { where: 'to the left', other: box(60, 100, 40, 20) },
    { where: 'below', other: box(100, 120, 40, 20) },
    { where: 'above', other: box(100, 80, 40, 20) },
  ])('are clear where the other touches at $where', ({ other }) => {
    expect(areBoxesSeparate(one, other)).toBe(true)
  })

  it.each([
    { where: 'to the right', other: box(139, 100, 40, 20) },
    { where: 'to the left', other: box(61, 100, 40, 20) },
    { where: 'below', other: box(100, 119, 40, 20) },
    { where: 'above', other: box(100, 81, 40, 20) },
  ])('are not clear where the other overlaps at $where', ({ other }) => {
    expect(areBoxesSeparate(one, other)).toBe(false)
  })
})

describe('a number set against a height', () => {
  const y = 100

  it('is clear where nothing is drawn near it', () => {
    expect(isAxisLabelClear(y, [], [])).toBe(true)
  })

  it('is clear where a mark is missing', () => {
    expect(isAxisLabelClear(y, [], [null])).toBe(true)
  })

  it('is not clear of a mark at the edge of the room a number takes across', () => {
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT + AXIS_LABEL_WIDTH, y }])).toBe(false)
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT + AXIS_LABEL_WIDTH + 1, y }])).toBe(true)
  })

  it('is not clear of a mark at the edge of the room a number takes down', () => {
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT, y: y + AXIS_LABEL_HEIGHT }])).toBe(false)
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT, y: y - AXIS_LABEL_HEIGHT }])).toBe(false)
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT, y: y + AXIS_LABEL_HEIGHT + 1 }])).toBe(true)
  })

  it('is not clear of a point of the curve at the same edges', () => {
    expect(isAxisLabelClear(y, [{ x: PLOT_LEFT + AXIS_LABEL_WIDTH, y }], [])).toBe(false)
    expect(isAxisLabelClear(y, [{ x: PLOT_LEFT, y: y + AXIS_LABEL_HEIGHT }], [])).toBe(false)
  })
})

describe('where a name is pulled back at the ends', () => {
  it('is centred on a mark exactly a label from either end', () => {
    expect(getMarkNameStyle({ x: PLOT_LEFT + LABEL, y: 100 }).translate).toBe('-50% -100%')
    expect(getMarkNameStyle({ x: PLOT_RIGHT - LABEL, y: 100 }).translate).toBe('-50% -100%')
  })

  it('takes a centred room for a mark exactly a label from either end', () => {
    expect(getNameBox({ x: PLOT_LEFT + LABEL, y: 100 }).x).toBe(PLOT_LEFT)
    expect(getNameBox({ x: PLOT_RIGHT - LABEL, y: 100 }).x).toBe(PLOT_RIGHT - LABEL * 2)
  })

  it('is centred on the bubble of a knob exactly half a bubble from either end', () => {
    const half = CALLOUT_WIDTH / 2

    expect(createCallout({ x: PLOT_LEFT + half, y: 120 }).at.translate).toBe('-50% -100%')
    expect(createCallout({ x: PLOT_RIGHT - half, y: 120 }).at.translate).toBe('-50% -100%')
  })

  it('is centred on the value of a knob exactly a label from either end', () => {
    expect(getKnobValueStyle({ x: PLOT_LEFT + LABEL, y: 100 }).translate).toBe('-50% 0')
    expect(getKnobValueStyle({ x: PLOT_RIGHT - LABEL, y: 100 }).translate).toBe('-50% 0')
  })
})

describe('where the number over the top is set', () => {
  it('remains over the top line, lifted clear of it', () => {
    const numbers = createAxisLabels(
      { least: 0, most: 10 },
      { places: clear, marks: [] },
      formatValue,
    )

    expect(numbers[0]!.at).toStrictEqual(getAxisNumber(PLOT_TOP, '-100%'))
    expect(numbers[0]!.at.translate).toBe('0 -100%')
  })
})
