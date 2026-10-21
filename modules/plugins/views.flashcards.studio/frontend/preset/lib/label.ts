/**
 * The words set on the one control's picture, and the room each of them takes.
 *
 * Everything here is worked out in the picture's own units. A word is placed
 * against a line or over a mark, and what it evaluates to is a box; two boxes that
 * touch are two words that would print over each other, and the picture drops
 * one of them.
 */
import type { CSSProperties } from 'vue'
import type { Position } from '@numen/ui'
import {
  PLOT_BOTTOM,
  PLOT_HEIGHT,
  PLOT_LEFT,
  PLOT_RIGHT,
  PLOT_TOP,
  PLOT_WIDTH,
  type Extent,
} from './plot'

/** How far the name of a mark stands above it, and how wide it is set. */
export const AXIS_LABEL_LIFT = 9
export const LABEL = 34

/** The room a number set against a line takes, in the picture's own units. */
export const AXIS_LABEL_WIDTH = 64
export const AXIS_LABEL_HEIGHT = 15

/**
 * The room the bubble over the knob takes, and how far off the knob it sits.
 * The width and height are what it evaluates to at the interface's own size, and
 * are what everything else on the picture is kept clear of. The gap stands off
 * far enough that the curve on either side of the knob is read through it, and
 * is the same gap on whichever side the bubble hangs.
 */
export const CALLOUT_WIDTH = 104
export const CALLOUT_HEIGHT = 48
export const CALLOUT_GAP = 24

/** The room one word takes on the picture, in the picture's own units. */
export interface Box {
  readonly x: number
  readonly y: number
  readonly wide: number
  readonly high: number
}

/** Whether two words stand clear of each other. */
export const areBoxesSeparate = (one: Box, two: Box): boolean =>
  one.x + one.wide <= two.x ||
  two.x + two.wide <= one.x ||
  one.y + one.high <= two.y ||
  two.y + two.high <= one.y

/**
 * Whether a number set against this height stands clear of everything drawn
 * near the edge it is set at. A number that would print over the line or over
 * a mark is not drawn: the axis loses the label and the drawing keeps its own.
 */
export const isAxisLabelClear = (
  y: number,
  places: readonly Position[],
  marks: readonly (Position | null)[],
): boolean => {
  const isNear = (at: Position): boolean =>
    at.x <= PLOT_LEFT + AXIS_LABEL_WIDTH && Math.abs(at.y - y) <= AXIS_LABEL_HEIGHT
  if (marks.some((at) => at !== null && isNear(at))) return false
  return !places.some(isNear)
}

/**
 * Where a name over a mark is set: above it, pulled back inside the picture at
 * either end so the whole word stands over it.
 */
export const getMarkNameStyle = (at: Position): CSSProperties => {
  let back = '-50%'
  if (at.x < PLOT_LEFT + LABEL) back = '0'
  else if (at.x > PLOT_RIGHT - LABEL) back = '-100%'
  return {
    insetInlineStart: `${(at.x / PLOT_WIDTH) * 100}%`,
    insetBlockStart: `${(Math.max(at.y - AXIS_LABEL_LIFT, PLOT_TOP) / PLOT_HEIGHT) * 100}%`,
    translate: `${back} -100%`,
  }
}

/** The room that name takes, which the knob's own figures stand clear of. */
export const getNameBox = (at: Position): Box => {
  let back = LABEL
  if (at.x < PLOT_LEFT + LABEL) back = 0
  else if (at.x > PLOT_RIGHT - LABEL) back = LABEL * 2
  return {
    x: at.x - back,
    y: Math.max(at.y - AXIS_LABEL_LIFT, PLOT_TOP) - AXIS_LABEL_HEIGHT,
    wide: LABEL * 2,
    high: AXIS_LABEL_HEIGHT,
  }
}

/** Where a number against one of a plot's own lines is set, in the plot's room. */
export const getAxisNumber = (y: number, lift: string, high = PLOT_HEIGHT): CSSProperties => ({
  insetInlineStart: `${(PLOT_LEFT / PLOT_WIDTH) * 100}%`,
  insetBlockStart: `${(y / high) * 100}%`,
  translate: `0 ${lift}`,
})

/** The room that number takes, which the knob's own figures stand clear of. */
export const getAxisNumberBox = (y: number, lift: string): Box => ({
  x: PLOT_LEFT,
  y: lift === '0' ? y : y - AXIS_LABEL_HEIGHT,
  wide: AXIS_LABEL_WIDTH,
  high: AXIS_LABEL_HEIGHT,
})

/** The bubble over the knob: the side it hangs on, the room it takes, and where. */
export interface Callout {
  readonly isUnder: boolean
  readonly box: Box
  readonly at: CSSProperties
  readonly tail: CSSProperties
}

/**
 * Where the bubble over the knob is set. It sits above the knob, and below it
 * where above would take it off the top, so it never covers the curve the knob
 * is riding. The tail is anchored on the knob itself and turns over with the
 * bubble, so what the numbers belong to is never in doubt.
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

/** A mark the picture carries, and the name it wants over it where it wants one. */
export interface Mark {
  readonly key: string
  readonly at: Position
  readonly text: string
}

/** A name that fit: where it is set, and the room it took. */
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
export const createLabels = (marks: readonly Mark[], over: Box | null): readonly Label[] => {
  const placed: Box[] = over ? [over] : []
  const out: Label[] = []
  for (const mark of marks) {
    if (!mark.text) continue
    const box = getNameBox(mark.at)
    if (!placed.every((one) => areBoxesSeparate(box, one))) continue
    placed.push(box)
    out.push({ key: mark.key, text: mark.text, at: getMarkNameStyle(mark.at), box })
  }
  return out
}

/** A number read off an edge of the picture: where it is set, and what it says. */
export interface Height {
  readonly at: CSSProperties
  readonly box: Box
  readonly text: string
}

/**
 * What the height of the picture evaluates to, against the lines it is read off. An
 * extent of no width is one number and is said once, on the foot, where a curve
 * that never moves is drawn. A number the line, a mark or the readout over the
 * knob relies on is dropped: the axis gives way, and the drawing keeps what it.
 */
export interface AxisLabelsContext {
  places: readonly Position[]
  marks: readonly Position[]
  over?: Box | null
}

export const createAxisLabels = (
  extent: Extent,
  context: AxisLabelsContext,
  getText: (value: number) => string,
): readonly Height[] => {
  const { least, most } = extent
  const { places, marks, over = null } = context
  const measureHeightAt = (y: number, lift: string, value: number): readonly Height[] => {
    const box = getAxisNumberBox(y, lift)
    if (!isAxisLabelClear(y, places, marks)) return []
    if (over && !areBoxesSeparate(box, over)) return []
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
 * Where the knob's own value is set. It rides a line of its own under the
 * picture, so it never prints over a number read off the picture's edges.
 */
export const getKnobValueStyle = (knob: Position | null): CSSProperties => {
  if (!knob) return {}
  let back = '-50%'
  if (knob.x < PLOT_LEFT + LABEL) back = '0'
  else if (knob.x > PLOT_RIGHT - LABEL) back = '-100%'
  return { insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`, translate: `${back} 0` }
}
