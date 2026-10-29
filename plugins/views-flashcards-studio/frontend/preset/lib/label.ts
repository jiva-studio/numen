/**
 * The text labels of the curve control plot and the space each takes.
 *
 * All geometry is in plot units. A label is positioned against a line or above
 * a mark and evaluates to a box; two overlapping boxes would print over each
 * other, and the plot omits one of them.
 */
export interface InlinePositionStyle {
  readonly insetInlineStart?: string
  readonly insetBlockStart?: string
  readonly translate?: string
}
type CSSProperties = InlinePositionStyle
import type { Position } from '@numen/ui'
import {
  BACKLOG_PLOT,
  PLOT_BOTTOM,
  PLOT_HEIGHT,
  PLOT_LEFT,
  PLOT_RIGHT,
  PLOT_TOP,
  PLOT_WIDTH,
  type Extent,
} from './plot'

/** Vertical offset of a mark name above its mark, and half the width of a mark name. */
export const AXIS_LABEL_LIFT = 9
export const LABEL = 34

/** Space taken by a number placed against a line, in plot units. */
export const AXIS_LABEL_WIDTH = 64
export const AXIS_LABEL_HEIGHT = 15

/**
 * Space taken by the callout above the knob, and its distance from the knob.
 * The width and height are the rendered size at the interface's default size,
 * and every other plot element is kept clear of that box. The gap leaves the
 * curve on either side of the knob visible, and is the same on whichever side
 * the callout is placed.
 */
export const CALLOUT_WIDTH = 104
export const CALLOUT_HEIGHT = 48
export const CALLOUT_GAP = 24

/** Space taken by one label on the plot, in plot units. */
export interface Box {
  readonly x: number
  readonly y: number
  readonly wide: number
  readonly high: number
}

/** Whether two label boxes do not overlap. */
export const areBoxesSeparate = (one: Box, two: Box): boolean =>
  one.x + one.wide <= two.x ||
  two.x + two.wide <= one.x ||
  one.y + one.high <= two.y ||
  two.y + two.high <= one.y

/**
 * Whether a number placed at this height does not overlap anything drawn near
 * the edge it is placed at. A number that would overlap the line or a mark is
 * omitted: the axis drops the label and the drawing keeps its own.
 */
export const isAxisLabelClear = (
  y: number,
  places: readonly Position[],
  marks: readonly (Position | null)[],
): boolean => {
  const isNear = (position: Position): boolean =>
    position.x <= PLOT_LEFT + AXIS_LABEL_WIDTH && Math.abs(position.y - y) <= AXIS_LABEL_HEIGHT
  if (marks.some((pos) => pos !== null && isNear(pos))) return false
  return !places.some(isNear)
}

/**
 * Position of a mark name: above the mark, shifted inward where the mark is
 * near the left or right edge of the plot so the name stays inside the plot.
 */
export const getMarkNameStyle = (position: Position): CSSProperties => {
  let back = '-50%'
  if (position.x < PLOT_LEFT + LABEL) back = '0'
  else if (position.x > PLOT_RIGHT - LABEL) back = '-100%'
  return {
    insetInlineStart: `${(position.x / PLOT_WIDTH) * 100}%`,
    insetBlockStart: `${(Math.max(position.y - AXIS_LABEL_LIFT, PLOT_TOP) / PLOT_HEIGHT) * 100}%`,
    translate: `${back} -100%`,
  }
}

/** Space taken by a mark name, which the knob's own figures are kept clear of. */
export const getNameBox = (position: Position): Box => {
  let back = LABEL
  if (position.x < PLOT_LEFT + LABEL) back = 0
  else if (position.x > PLOT_RIGHT - LABEL) back = LABEL * 2
  return {
    x: position.x - back,
    y: Math.max(position.y - AXIS_LABEL_LIFT, PLOT_TOP) - AXIS_LABEL_HEIGHT,
    wide: LABEL * 2,
    high: AXIS_LABEL_HEIGHT,
  }
}

/** Position of a number against one of a plot's own lines, in the plot's space. */
export const getAxisNumber = (y: number, lift: string, high = PLOT_HEIGHT): CSSProperties => ({
  insetInlineStart: `${(PLOT_LEFT / PLOT_WIDTH) * 100}%`,
  insetBlockStart: `${(y / high) * 100}%`,
  translate: `0 ${lift}`,
})

/** Space taken by that number, which the knob's own figures are kept clear of. */
export const getAxisNumberBox = (y: number, lift: string): Box => ({
  x: PLOT_LEFT,
  y: lift === '0' ? y : y - AXIS_LABEL_HEIGHT,
  wide: AXIS_LABEL_WIDTH,
  high: AXIS_LABEL_HEIGHT,
})

/** The callout above the knob: the side it is placed on, the space it takes, and its position. */
export interface Callout {
  readonly isUnder: boolean
  readonly box: Box
  readonly at: CSSProperties
  readonly tail: CSSProperties
}

/**
 * Position of the callout above the knob. It is placed above the knob, and
 * below it where above would leave the top of the plot, so it never covers the
 * curve the knob is on. The tail is anchored on the knob and flips with the
 * callout, so the numbers always refer to the knob.
 */
export const createCallout = (knob: Position): Callout => {
  const isUnder = knob.y - CALLOUT_GAP - CALLOUT_HEIGHT < PLOT_TOP
  const half = CALLOUT_WIDTH / 2
  let back = half
  if (knob.x < PLOT_LEFT + half) back = 0
  else if (knob.x > PLOT_RIGHT - half) back = CALLOUT_WIDTH
  const edge = isUnder ? knob.y + CALLOUT_GAP : knob.y - CALLOUT_GAP
  return {
    isUnder,
    box: {
      x: knob.x - back,
      y: isUnder ? edge : edge - CALLOUT_HEIGHT,
      wide: CALLOUT_WIDTH,
      high: CALLOUT_HEIGHT,
    },
    at: {
      insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`,
      insetBlockStart: `${(edge / PLOT_HEIGHT) * 100}%`,
      translate: `${(-back / CALLOUT_WIDTH) * 100}% ${isUnder ? '0' : '-100%'}`,
    },
    tail: {
      insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`,
      insetBlockStart: `${(edge / PLOT_HEIGHT) * 100}%`,
    },
  }
}

/** A mark on the plot, and the name displayed above it where it has one. */
export interface Mark {
  readonly key: string
  readonly at: Position
  readonly text: string
}

/** A mark name that fits: its position and the space it takes. */
export interface Label {
  readonly key: string
  readonly text: string
  readonly at: CSSProperties
  readonly box: Box
}

/**
 * The names that fit. They are taken in the order the marks stand in, and one
 * that would touch the readout over the knob, or a name already placed, is
 * left off.
 */
export const createLabels = (marks: readonly Mark[], calloutBox: Box | null): readonly Label[] => {
  const occupiedBoxes: Box[] = calloutBox ? [calloutBox] : []
  const out: Label[] = []
  for (const mark of marks) {
    if (!mark.text) continue
    const box = getNameBox(mark.at)
    if (!occupiedBoxes.every((one) => areBoxesSeparate(box, one))) continue
    occupiedBoxes.push(box)
    out.push({ key: mark.key, text: mark.text, at: getMarkNameStyle(mark.at), box })
  }
  return out
}

/** A number read at an edge of the plot: its position and its text. */
export interface Height {
  readonly at: CSSProperties
  readonly box: Box
  readonly text: string
}

/**
 * Axis labels for the plot height, positioned against the lines they are read
 * from. An extent of zero width is one number, displayed once at the foot, where
 * a constant curve is drawn. A number that overlaps the line, a mark or the
 * callout above the knob is omitted: the axis drops the label and the drawing
 * keeps its own.
 */
export interface AxisLabelsContext {
  places: readonly Position[]
  marks: readonly Position[]
  calloutBox?: Box | null
}

export const createAxisLabels = (
  extent: Extent,
  context: AxisLabelsContext,
  getText: (value: number) => string,
): readonly Height[] => {
  const { min: least, max: most } = extent
  const { places, marks, calloutBox = null } = context
  const measureHeightAt = (y: number, lift: string, value: number): readonly Height[] => {
    const box = getAxisNumberBox(y, lift)
    if (!isAxisLabelClear(y, places, marks)) return []
    if (calloutBox && !areBoxesSeparate(box, calloutBox)) return []
    return [{ at: getAxisNumber(y, lift), box, text: getText(value) }]
  }
  // An extent of no width has one number and nothing else to read, and it is set
  // over the line it names. That line is the foot, which is where a run with no
  // height is drawn.
  if (most === least) {
    return [
      {
        at: getAxisNumber(PLOT_BOTTOM, '0'),
        box: getAxisNumberBox(PLOT_BOTTOM, '0'),
        text: getText(most),
      },
    ]
  }
  return [...measureHeightAt(PLOT_TOP, '-100%', most), ...measureHeightAt(PLOT_BOTTOM, '0', least)]
}

/**
 * Computes the axis height labels for the backlog plot.
 */
export const createBacklogAxisLabels = (
  extent: Extent,
  places: readonly Position[],
  formatValue: (value: number) => string,
): readonly { at: CSSProperties; text: string }[] => {
  const { min: least, max: most } = extent
  const createLabel = (y: number, lift: string, value: number) =>
    isAxisLabelClear(y, places, [])
      ? [{ at: getAxisNumber(y, lift, BACKLOG_PLOT.high), text: formatValue(value) }]
      : []
  if (most === least) {
    return [
      { at: getAxisNumber(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high), text: formatValue(least) },
    ]
  }
  return [
    ...createLabel(BACKLOG_PLOT.top, '-100%', most),
    ...createLabel(BACKLOG_PLOT.foot, '0', least),
  ]
}

/**
 * Position of the knob's own value. It is placed on a line of its own below the
 * plot, so it never overlaps a number read at the plot edges.
 */
export const getKnobValueStyle = (knob: Position | null): CSSProperties => {
  if (!knob) return {}
  let back = '-50%'
  if (knob.x < PLOT_LEFT + LABEL) back = '0'
  else if (knob.x > PLOT_RIGHT - LABEL) back = '-100%'
  return { insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`, translate: `${back} 0` }
}
