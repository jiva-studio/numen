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

const box = (x: number, y: number, width: number, height: number): Box => ({
  x,
  y,
  wide: width,
  high: height,
})

const isInside = (one: Box): boolean =>
  one.x >= 0 && one.y >= 0 && one.x + one.wide <= PLOT_WIDTH && one.y + one.high <= PLOT_HEIGHT

describe('getMarkNameStyle', () => {
  it('positions the label above the mark', () => {
    const style = getMarkNameStyle({ x: PLOT_WIDTH / 2, y: 100 })

    expect(style.insetInlineStart).toBe('50%')
    expect(style.insetBlockStart).toBe(`${((100 - AXIS_LABEL_LIFT) / PLOT_HEIGHT) * 100}%`)
    expect(style.translate).toBe('-50% -100%')
  })

  it('aligns the label inside the plot at the left and right edges', () => {
    expect(getMarkNameStyle({ x: PLOT_LEFT, y: 100 }).translate).toBe('0 -100%')
    expect(getMarkNameStyle({ x: PLOT_RIGHT, y: 100 }).translate).toBe('-100% -100%')
  })

  it('clamps the label at the top edge of the plot', () => {
    expect(getMarkNameStyle({ x: PLOT_WIDTH / 2, y: PLOT_TOP }).insetBlockStart).toBe(
      `${(PLOT_TOP / PLOT_HEIGHT) * 100}%`,
    )
  })
})

describe('getNameBox', () => {
  it('returns the bounding box centered on the mark', () => {
    const room = getNameBox({ x: PLOT_WIDTH / 2, y: 100 })

    expect(room).toStrictEqual({
      x: PLOT_WIDTH / 2 - LABEL,
      y: 100 - AXIS_LABEL_LIFT - AXIS_LABEL_HEIGHT,
      wide: LABEL * 2,
      high: AXIS_LABEL_HEIGHT,
    })
  })

  it('shifts the box inside the plot at the left and right edges', () => {
    expect(getNameBox({ x: PLOT_LEFT, y: 100 }).x).toBe(PLOT_LEFT)
    expect(getNameBox({ x: PLOT_RIGHT, y: 100 }).x).toBe(PLOT_RIGHT - LABEL * 2)
  })
})

describe('getAxisNumber', () => {
  it('positions the axis number at the left edge and given height', () => {
    const style = getAxisNumber(PLOT_BOTTOM, '0')

    expect(style.insetInlineStart).toBe(`${(PLOT_LEFT / PLOT_WIDTH) * 100}%`)
    expect(style.insetBlockStart).toBe(`${(PLOT_BOTTOM / PLOT_HEIGHT) * 100}%`)
    expect(style.translate).toBe('0 0')
  })

  it('scales the vertical position by the custom plot height', () => {
    expect(getAxisNumber(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high).insetBlockStart).toBe(
      `${(BACKLOG_PLOT.foot / BACKLOG_PLOT.high) * 100}%`,
    )
  })
})

describe('getAxisNumberBox', () => {
  it('places the box below y when offsetY is 0', () => {
    expect(getAxisNumberBox(PLOT_BOTTOM, '0')).toStrictEqual({
      x: PLOT_LEFT,
      y: PLOT_BOTTOM,
      wide: AXIS_LABEL_WIDTH,
      high: AXIS_LABEL_HEIGHT,
    })
  })

  it('places the box above y when offsetY is negative', () => {
    expect(getAxisNumberBox(PLOT_TOP, '-100%').y).toBe(PLOT_TOP - AXIS_LABEL_HEIGHT)
  })
})

describe('createCallout', () => {
  it('places the callout above the knob by the gap', () => {
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

  it('places the callout below the knob near the top edge', () => {
    const callout = createCallout({ x: PLOT_WIDTH / 2, y: PLOT_TOP })

    expect(callout.isUnder).toBe(true)
    expect(callout.box.y).toBe(PLOT_TOP + CALLOUT_GAP)
    expect(callout.at.translate).toBe('-50% 0')
  })

  it('flips the callout at the top boundary threshold', () => {
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
  ])('keeps the callout inside the plot at $where', ({ knob, back }) => {
    const callout = createCallout(knob)

    expect(callout.at.translate).toBe(`${back} -100%`)
    expect(isInside(callout.box)).toBe(true)
  })

  it('anchors the callout tail on the knob', () => {
    const callout = createCallout({ x: PLOT_LEFT, y: 120 })

    expect(callout.tail.insetInlineStart).toBe(callout.at.insetInlineStart)
    expect(callout.tail.insetBlockStart).toBe(callout.at.insetBlockStart)
  })
})

describe('createLabels', () => {
  it('returns an empty list when there are no marks', () => {
    expect(createLabels([], null)).toStrictEqual([])
  })

  it('creates a label for a single mark', () => {
    expect(createLabels([mark('suggested', PLOT_WIDTH / 2, 100)], null)).toStrictEqual([
      {
        key: 'suggested',
        text: 'suggested',
        at: getMarkNameStyle({ x: PLOT_WIDTH / 2, y: 100 }),
        box: getNameBox({ x: PLOT_WIDTH / 2, y: 100 }),
      },
    ])
  })

  it('skips marks with empty text', () => {
    expect(createLabels([mark('knob', PLOT_WIDTH / 2, 100, '')], null)).toStrictEqual([])
  })

  it.each([
    { gap: LABEL * 2 - 1, expected: ['first'] },
    { gap: LABEL * 2, expected: ['first', 'second'] },
    { gap: LABEL * 2 + 1, expected: ['first', 'second'] },
  ])('omits overlapping labels when separated by $gap', ({ gap, expected }) => {
    const names = createLabels([mark('first', 200, 100), mark('second', 200 + gap, 100)], null)

    expect(names.map((one) => one.key)).toStrictEqual(expected)
  })

  it('keeps two labels at the same x when their y positions do not overlap', () => {
    const names = createLabels(
      [mark('first', 200, 60), mark('second', 200, 60 + AXIS_LABEL_LIFT + AXIS_LABEL_HEIGHT)],
      null,
    )

    expect(names.map((one) => one.key)).toStrictEqual(['first', 'second'])
  })

  it('omits a label that overlaps the callout box', () => {
    const calloutBox = createCallout({ x: 280, y: PLOT_TOP }).box

    expect(createLabels([mark('suggested', 280, 100)], calloutBox)).toStrictEqual([])
    expect(createLabels([mark('suggested', 280, 100)], null)).toHaveLength(1)
  })

  it.each([
    { where: 'the left edge', at: { x: PLOT_LEFT, y: 100 } },
    { where: 'the right edge', at: { x: PLOT_RIGHT, y: 100 } },
    { where: 'the top', at: { x: PLOT_WIDTH / 2, y: PLOT_TOP } },
    { where: 'the foot', at: { x: PLOT_WIDTH / 2, y: PLOT_BOTTOM } },
  ])('keeps the label box inside the plot at $where', ({ at: pos }) => {
    const names = createLabels([mark('suggested', pos.x, pos.y)], null)

    expect(names).toHaveLength(1)
    expect(isInside(names[0]!.box)).toBe(true)
  })
})

const formatValue = (value: number): string => String(value)

const clear: readonly Position[] = [
  { x: 300, y: PLOT_TOP },
  { x: PLOT_RIGHT, y: PLOT_BOTTOM },
]

describe('createAxisLabels', () => {
  it('places the max label at the top and the min label at the bottom', () => {
    const numbers = createAxisLabels({ min: 0, max: 10 }, { places: clear, marks: [] }, formatValue)

    expect(numbers.map((one) => one.text)).toStrictEqual(['10', '0'])
    expect(numbers.map((one) => one.box)).toStrictEqual([
      getAxisNumberBox(PLOT_TOP, '-100%'),
      getAxisNumberBox(PLOT_BOTTOM, '0'),
    ])
  })

  it('places a single label at the bottom when min equals max', () => {
    const numbers = createAxisLabels({ min: 4, max: 4 }, { places: clear, marks: [] }, formatValue)

    expect(numbers).toStrictEqual([
      { at: getAxisNumber(PLOT_BOTTOM, '0'), box: getAxisNumberBox(PLOT_BOTTOM, '0'), text: '4' },
    ])
  })

  it('omits an axis label that overlaps a curve point', () => {
    const numbers = createAxisLabels(
      { min: 0, max: 10 },
      { places: [{ x: PLOT_LEFT, y: PLOT_BOTTOM }], marks: [] },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['10'])
  })

  it('omits an axis label that overlaps a mark', () => {
    const numbers = createAxisLabels(
      { min: 0, max: 10 },
      { places: clear, marks: [{ x: PLOT_LEFT, y: PLOT_TOP }] },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('omits an axis label that overlaps the callout box', () => {
    const calloutBox = box(PLOT_LEFT, 0, CALLOUT_WIDTH, CALLOUT_HEIGHT)
    const numbers = createAxisLabels(
      { min: 0, max: 10 },
      { places: clear, marks: [], calloutBox },
      formatValue,
    )

    expect(numbers.map((one) => one.text)).toStrictEqual(['0'])
  })

  it('keeps all axis label boxes inside the plot', () => {
    const numbers = createAxisLabels({ min: 0, max: 10 }, { places: clear, marks: [] }, formatValue)

    expect(numbers.every((one) => isInside(one.box))).toBe(true)
  })
})

describe('getKnobValueStyle', () => {
  it('returns an empty style when knob is null', () => {
    expect(getKnobValueStyle(null)).toStrictEqual({})
  })

  it.each([
    { where: 'the left edge', x: PLOT_LEFT, back: '0' },
    { where: 'the middle', x: PLOT_WIDTH / 2, back: '-50%' },
    { where: 'the right edge', x: PLOT_RIGHT, back: '-100%' },
  ])('aligns the knob value inside the plot at $where', ({ x, back }) => {
    const style = getKnobValueStyle({ x, y: 100 })

    expect(style.insetInlineStart).toBe(`${(x / PLOT_WIDTH) * 100}%`)
    expect(style.translate).toBe(`${back} 0`)
  })
})

describe('areBoxesSeparate', () => {
  const one = box(100, 100, 40, 20)

  it.each([
    { where: 'to the right', other: box(140, 100, 40, 20) },
    { where: 'to the left', other: box(60, 100, 40, 20) },
    { where: 'below', other: box(100, 120, 40, 20) },
    { where: 'above', other: box(100, 80, 40, 20) },
  ])('returns true when boxes touch at $where', ({ other }) => {
    expect(areBoxesSeparate(one, other)).toBe(true)
  })

  it.each([
    { where: 'to the right', other: box(139, 100, 40, 20) },
    { where: 'to the left', other: box(61, 100, 40, 20) },
    { where: 'below', other: box(100, 119, 40, 20) },
    { where: 'above', other: box(100, 81, 40, 20) },
  ])('returns false when boxes overlap at $where', ({ other }) => {
    expect(areBoxesSeparate(one, other)).toBe(false)
  })
})

describe('isAxisLabelClear', () => {
  const y = 100

  it('returns true when no points or marks are near', () => {
    expect(isAxisLabelClear(y, [], [])).toBe(true)
  })

  it('returns true when a mark is null', () => {
    expect(isAxisLabelClear(y, [], [null])).toBe(true)
  })

  it('returns false when a mark is within the label width', () => {
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT + AXIS_LABEL_WIDTH, y }])).toBe(false)
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT + AXIS_LABEL_WIDTH + 1, y }])).toBe(true)
  })

  it('returns false when a mark is within the label height', () => {
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT, y: y + AXIS_LABEL_HEIGHT }])).toBe(false)
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT, y: y - AXIS_LABEL_HEIGHT }])).toBe(false)
    expect(isAxisLabelClear(y, [], [{ x: PLOT_LEFT, y: y + AXIS_LABEL_HEIGHT + 1 }])).toBe(true)
  })

  it('returns false when a curve point is within the label bounds', () => {
    expect(isAxisLabelClear(y, [{ x: PLOT_LEFT + AXIS_LABEL_WIDTH, y }], [])).toBe(false)
    expect(isAxisLabelClear(y, [{ x: PLOT_LEFT, y: y + AXIS_LABEL_HEIGHT }], [])).toBe(false)
  })
})

describe('edge alignment thresholds', () => {
  it('centers the mark name when one label width from either edge', () => {
    expect(getMarkNameStyle({ x: PLOT_LEFT + LABEL, y: 100 }).translate).toBe('-50% -100%')
    expect(getMarkNameStyle({ x: PLOT_RIGHT - LABEL, y: 100 }).translate).toBe('-50% -100%')
  })

  it('centers the mark box when one label width from either edge', () => {
    expect(getNameBox({ x: PLOT_LEFT + LABEL, y: 100 }).x).toBe(PLOT_LEFT)
    expect(getNameBox({ x: PLOT_RIGHT - LABEL, y: 100 }).x).toBe(PLOT_RIGHT - LABEL * 2)
  })

  it('centers the callout when half a callout width from either edge', () => {
    const half = CALLOUT_WIDTH / 2

    expect(createCallout({ x: PLOT_LEFT + half, y: 120 }).at.translate).toBe('-50% -100%')
    expect(createCallout({ x: PLOT_RIGHT - half, y: 120 }).at.translate).toBe('-50% -100%')
  })

  it('centers the knob value when one label width from either edge', () => {
    expect(getKnobValueStyle({ x: PLOT_LEFT + LABEL, y: 100 }).translate).toBe('-50% 0')
    expect(getKnobValueStyle({ x: PLOT_RIGHT - LABEL, y: 100 }).translate).toBe('-50% 0')
  })
})

describe('top axis number position', () => {
  it('positions the top axis number above the top line', () => {
    const numbers = createAxisLabels({ min: 0, max: 10 }, { places: clear, marks: [] }, formatValue)

    expect(numbers[0]!.at).toStrictEqual(getAxisNumber(PLOT_TOP, '-100%'))
    expect(numbers[0]!.at.translate).toBe('0 -100%')
  })
})
