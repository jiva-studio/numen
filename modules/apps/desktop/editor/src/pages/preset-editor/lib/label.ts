/**
 * The words set on the one control's picture, and the room each of them takes.
 *
 * Everything here is worked out in the picture's own units. A word is placed
 * against a line or over a mark, and what it comes to is a box; two boxes that
 * touch are two words that would print over each other, and the picture drops
 * one of them.
 */
import type { CSSProperties } from 'vue'
import type { Position } from '@numen/ui'
import { FOOT, HIGH, LEFT, RIGHT, TOP, WIDE, type Extent } from './plot'

/** How far the name of a mark stands above it, and how wide it is set. */
export const LIFT = 9
export const LABEL = 34

/** The room a number set against a line takes, in the picture's own units. */
export const AXIS_WIDE = 64
export const AXIS_HIGH = 15

/**
 * The room the bubble over the knob takes, and how far off the knob it sits.
 * The width and height are what it comes to at the interface's own size, and
 * are what everything else on the picture is kept clear of. The gap stands off
 * far enough that the curve on either side of the knob is read through it, and
 * is the same gap on whichever side the bubble hangs.
 */
export const CALLOUT_WIDE = 104
export const CALLOUT_HIGH = 48
export const CALLOUT_GAP = 24

/** The room one word takes on the picture, in the picture's own units. */
export interface Box {
  readonly x: number
  readonly y: number
  readonly wide: number
  readonly high: number
}

/** Whether two words stand clear of each other. */
export const isApart = (one: Box, two: Box): boolean =>
  one.x + one.wide <= two.x ||
  two.x + two.wide <= one.x ||
  one.y + one.high <= two.y ||
  two.y + two.high <= one.y

/**
 * Whether a number set against this height stands clear of everything drawn
 * near the edge it is set at. A number that would print over the line or over
 * a mark is not drawn: the axis loses the label and the drawing keeps its own.
 */
export const clearAt = (
  y: number,
  places: readonly Position[],
  marks: readonly (Position | null)[],
): boolean => {
  const isNear = (at: Position): boolean =>
    at.x <= LEFT + AXIS_WIDE && Math.abs(at.y - y) <= AXIS_HIGH
  if (marks.some((at) => at !== null && isNear(at))) return false
  return !places.some(isNear)
}

/**
 * Where a name over a mark is set: above it, pulled back inside the picture at
 * either end so the whole word stands over it.
 */
export const positionLabel = (at: Position): CSSProperties => {
  const back = at.x < LEFT + LABEL ? '0' : at.x > RIGHT - LABEL ? '-100%' : '-50%'
  return {
    insetInlineStart: `${(at.x / WIDE) * 100}%`,
    insetBlockStart: `${(Math.max(at.y - LIFT, TOP) / HIGH) * 100}%`,
    translate: `${back} -100%`,
  }
}

/** The room that name takes, which the knob's own figures stand clear of. */
export const getNameBox = (at: Position): Box => {
  const back = at.x < LEFT + LABEL ? 0 : at.x > RIGHT - LABEL ? LABEL * 2 : LABEL
  return {
    x: at.x - back,
    y: Math.max(at.y - LIFT, TOP) - AXIS_HIGH,
    wide: LABEL * 2,
    high: AXIS_HIGH,
  }
}

/** Where a number against one of a plot's own lines is set, in the plot's room. */
export const getAxisNumber = (y: number, lift: string, high = HIGH): CSSProperties => ({
  insetInlineStart: `${(LEFT / WIDE) * 100}%`,
  insetBlockStart: `${(y / high) * 100}%`,
  translate: `0 ${lift}`,
})

/** The room that number takes, which the knob's own figures stand clear of. */
export const getAxisNumberBox = (y: number, lift: string): Box => ({
  x: LEFT,
  y: lift === '0' ? y : y - AXIS_HIGH,
  wide: AXIS_WIDE,
  high: AXIS_HIGH,
})

/** The bubble over the knob: the side it hangs on, the room it takes, and where. */
export interface Callout {
  readonly under: boolean
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
export const calloutOf = (knob: Position): Callout => {
  const under = knob.y - CALLOUT_GAP - CALLOUT_HIGH < TOP
  const half = CALLOUT_WIDE / 2
  const back = knob.x < LEFT + half ? 0 : knob.x > RIGHT - half ? CALLOUT_WIDE : half
  const edge = under ? knob.y + CALLOUT_GAP : knob.y - CALLOUT_GAP
  return {
    under,
    box: {
      x: knob.x - back,
      y: under ? edge : edge - CALLOUT_HIGH,
      wide: CALLOUT_WIDE,
      high: CALLOUT_HIGH,
    },
    at: {
      insetInlineStart: `${(knob.x / WIDE) * 100}%`,
      insetBlockStart: `${(edge / HIGH) * 100}%`,
      translate: `${(-back / CALLOUT_WIDE) * 100}% ${under ? '0' : '-100%'}`,
    },
    tail: {
      insetInlineStart: `${(knob.x / WIDE) * 100}%`,
      insetBlockStart: `${(edge / HIGH) * 100}%`,
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
export const labelsOf = (marks: readonly Mark[], over: Box | null): readonly Label[] => {
  const placed: Box[] = over ? [over] : []
  const out: Label[] = []
  for (const mark of marks) {
    if (!mark.text) continue
    const box = getNameBox(mark.at)
    if (!placed.every((one) => isApart(box, one))) continue
    placed.push(box)
    out.push({ key: mark.key, text: mark.text, at: positionLabel(mark.at), box })
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
 * What the height of the picture comes to, against the lines it is read off. A
 * extent of no width is one number and is said once, on the foot, where a curve
 * that never moves is drawn. A number the line, a mark or the readout over the
 * knob stands on is dropped: the axis gives way, and the drawing keeps what it
 * has to say.
 */
export const heightsOf = (
  extent: Extent,
  places: readonly Position[],
  marks: readonly Position[],
  over: Box | null,
  getText: (value: number) => string,
): readonly Height[] => {
  const { least, most } = extent
  const measureHeightAt = (y: number, lift: string, value: number): readonly Height[] => {
    const box = getAxisNumberBox(y, lift)
    if (!clearAt(y, places, marks)) return []
    if (over && !isApart(box, over)) return []
    return [{ at: getAxisNumber(y, lift), box, text: getText(value) }]
  }
  // An extent of no width has one number and nothing else to read, and it is set
  // over the line it names. That line is the foot, which is where a run with no
  // height is drawn.
  if (most === least) {
    return [{ at: getAxisNumber(FOOT, '0'), box: getAxisNumberBox(FOOT, '0'), text: getText(most) }]
  }
  return [...measureHeightAt(TOP, '-100%', most), ...measureHeightAt(FOOT, '0', least)]
}

/**
 * Where the knob's own value is set. It rides a line of its own under the
 * picture, so it never prints over a number read off the picture's edges.
 */
export const readingAt = (knob: Position | null): CSSProperties => {
  if (!knob) return {}
  const back = knob.x < LEFT + LABEL ? '0' : knob.x > RIGHT - LABEL ? '-100%' : '-50%'
  return { insetInlineStart: `${(knob.x / WIDE) * 100}%`, translate: `${back} 0` }
}
