/**
 * Where the one control's curve is drawn, without drawing it.
 *
 * Everything is worked out in the picture's own units and scaled to whatever
 * room the tab has. The curve is the control, so the same arithmetic that puts
 * a point on the line reads a place off a pointer.
 */
import type { CSSProperties } from 'vue'
import type { Point } from '@numen/ui'
import { costOf } from './curve'
import type { Curve } from './core'

/** The picture's own units. It is scaled whole, so a circle stays one. */
export const WIDE = 560
export const HIGH = 190

/** The room the line is drawn in, inside the picture. */
export const LEFT = 12
export const RIGHT = WIDE - 12
export const TOP = 16
export const FOOT = HIGH - 16

/** The height a plot's line is drawn between, and the picture it stands in. */
export interface Plot {
  readonly high: number
  readonly top: number
  readonly foot: number
}

/** The curve's own plot, which is the picture. */
export const CURVE_PLOT: Plot = { high: HIGH, top: TOP, foot: FOOT }

/** The backlog's plot under it, which is short and shares the picture's width. */
export const BACKLOG_HIGH = 76
export const BACKLOG_PLOT: Plot = { high: BACKLOG_HIGH, top: 12, foot: BACKLOG_HIGH - 12 }

/** The faint lines drawn across the picture, as shares of the plot's height. */
export const GRIDLINES: readonly number[] = [0.25, 0.5, 0.75]

/** How far the name of a mark stands above it, and how wide it is set. */
export const LIFT = 9
export const LABEL = 34

/** One place of the curve, where the picture draws it. */
export type { Point }

/** Where one place of a grid stands across the picture. */
export const xOf = (place: number, places: number): number =>
  places <= 1 ? LEFT : LEFT + ((RIGHT - LEFT) * place) / (places - 1)

/** The height of one of the faint lines. */
export const yOfGridline = (share: number): number => FOOT - (FOOT - TOP) * share

/** The stretch of cost the picture is scaled to. */
export interface Extent {
  readonly least: number
  readonly most: number
}

/**
 * The extent a curve is drawn against: nothing at the foot, and the most it
 * ever costs over the top. A count does not go below nothing, so the foot of
 * the picture is nothing under every run and a height is read against it.
 */
export const extentOf = (curve: Curve): Extent => {
  const costs = curve.at.map((one) => costOf(curve.goal, one))
  return { least: 0, most: costs.length === 0 ? 0 : Math.max(0, ...costs) }
}

/**
 * The extent a backlog is drawn against: nothing overdue at the foot, and the
 * most this pace ever stands at over the top. It is scaled to the one place
 * the knob stands at, so a day too short to carry what falls due is drawn
 * climbing over the height it has.
 */
export const extentOfBacklog = (backlog: readonly number[]): Extent => ({
  least: 0,
  most: backlog.length === 0 ? 0 : Math.max(...backlog),
})

/**
 * Where a backlog is drawn. Nothing overdue is the foot under every run, so a
 * run holding nothing at all lies along the floor and a run holding anything
 * is measured up from it.
 */
export const backlogPointsOf = (values: readonly number[], extent: Extent): readonly Point[] =>
  seriesOf(values, extent, BACKLOG_PLOT)

/**
 * Where every place of the curve is drawn, in the extent the picture is scaled
 * to. The extent is the goal's and not this answer's, so a curve that is flat
 * within it is drawn flat, and an extent of no width lies along the foot.
 */
export const pointsOf = (curve: Curve, extent: Extent): readonly Point[] =>
  seriesOf(
    curve.at.map((one) => costOf(curve.goal, one)),
    extent,
  )

/**
 * Where any reading of the curve is drawn: one figure a place, over the same
 * width and against the extent it is scaled to. The line is one such reading
 * and a second series — a backlog under it, say — is another, so a picture that
 * carries two carries them by the same arithmetic.
 */
export const seriesOf = (
  values: readonly number[],
  extent: Extent,
  plot: Plot = CURVE_PLOT,
): readonly Point[] => {
  const span = extent.most - extent.least
  // An extent of no width has no height to read: every value in it is the foot
  // of the plot, and the foot is where it is drawn.
  return values.map((value, place) => ({
    x: xOf(place, values.length),
    y:
      span > 0
        ? held(plot.foot - (plot.foot - plot.top) * ((value - extent.least) / span), plot)
        : plot.foot,
  }))
}

/** A height inside the plot the line is drawn in. */
const held = (y: number, plot: Plot): number => Math.min(Math.max(y, plot.top), plot.foot)

/** The line through those places. */
export const lineOf = (points: readonly Point[]): string =>
  points.map((point, at) => `${at === 0 ? 'M' : 'L'}${round(point.x)} ${round(point.y)}`).join(' ')

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
export const apart = (one: Box, two: Box): boolean =>
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
  points: readonly Point[],
  marks: readonly (Point | null)[],
): boolean => {
  const near = (point: Point): boolean =>
    point.x <= LEFT + AXIS_WIDE && Math.abs(point.y - y) <= AXIS_HIGH
  if (marks.some((point) => point !== null && near(point))) return false
  return !points.some(near)
}

/**
 * The stretch the budget the preset keeps does not get through, drawn as a
 * line of its own. A goal that keeps no such account leaves it empty.
 */
export const shortOf = (curve: Curve, points: readonly Point[]): string => {
  const runs: string[] = []
  let run: Point[] = []
  curve.at.forEach((one, place) => {
    const point = points[place]
    if (!point) return
    if (one.enough) {
      if (run.length > 1) runs.push(lineOf(run))
      run = []
      return
    }
    run.push(point)
  })
  if (run.length > 1) runs.push(lineOf(run))
  return runs.join(' ')
}

/**
 * The days one place of the curve is drawn over. A goal of a date schedules
 * nothing past the day it names, so what the run says after that day is the
 * arithmetic of doing nothing and is no part of the choice being made.
 */
const daysAt = (curve: Curve, place: number): number =>
  curve.goal === 'date' ? Math.max(Math.round(curve.grid[place] ?? 0), 0) : -1

/** What stands overdue day by day at one place, cut at that place's own day. */
export const runAt = (curve: Curve, place: number): readonly number[] => {
  const run = curve.at[place]?.backlog ?? []
  const days = daysAt(curve, place)
  return days < 0 ? run : run.slice(0, days)
}

/**
 * Where a name over a mark is set: above it, pulled back inside the picture at
 * either end so the whole word stands over it.
 */
export const naming = (point: Point): CSSProperties => {
  const back = point.x < LEFT + LABEL ? '0' : point.x > RIGHT - LABEL ? '-100%' : '-50%'
  return {
    insetInlineStart: `${(point.x / WIDE) * 100}%`,
    insetBlockStart: `${(Math.max(point.y - LIFT, TOP) / HIGH) * 100}%`,
    translate: `${back} -100%`,
  }
}

/** The room that name takes, which the knob's own figures stand clear of. */
export const namingBox = (point: Point): Box => {
  const back = point.x < LEFT + LABEL ? 0 : point.x > RIGHT - LABEL ? LABEL * 2 : LABEL
  return {
    x: point.x - back,
    y: Math.max(point.y - LIFT, TOP) - AXIS_HIGH,
    wide: LABEL * 2,
    high: AXIS_HIGH,
  }
}

/** Where a number against one of a plot's own lines is set, in the plot's room. */
export const against = (y: number, lift: string, high = HIGH): CSSProperties => ({
  insetInlineStart: `${(LEFT / WIDE) * 100}%`,
  insetBlockStart: `${(y / high) * 100}%`,
  translate: `0 ${lift}`,
})

/** The room that number takes, which the knob's own figures stand clear of. */
export const againstBox = (y: number, lift: string): Box => ({
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
export const calloutOf = (point: Point): Callout => {
  const under = point.y - CALLOUT_GAP - CALLOUT_HIGH < TOP
  const half = CALLOUT_WIDE / 2
  const back = point.x < LEFT + half ? 0 : point.x > RIGHT - half ? CALLOUT_WIDE : half
  const edge = under ? point.y + CALLOUT_GAP : point.y - CALLOUT_GAP
  return {
    under,
    box: {
      x: point.x - back,
      y: under ? edge : edge - CALLOUT_HIGH,
      wide: CALLOUT_WIDE,
      high: CALLOUT_HIGH,
    },
    at: {
      insetInlineStart: `${(point.x / WIDE) * 100}%`,
      insetBlockStart: `${(edge / HIGH) * 100}%`,
      translate: `${(-back / CALLOUT_WIDE) * 100}% ${under ? '0' : '-100%'}`,
    },
    tail: {
      insetInlineStart: `${(point.x / WIDE) * 100}%`,
      insetBlockStart: `${(edge / HIGH) * 100}%`,
    },
  }
}

/** A mark the picture carries, and the name it wants over it where it wants one. */
export interface Mark {
  readonly key: string
  readonly point: Point
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
    const box = namingBox(mark.point)
    if (!placed.every((one) => apart(box, one))) continue
    placed.push(box)
    out.push({ key: mark.key, text: mark.text, at: naming(mark.point), box })
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
  points: readonly Point[],
  marks: readonly Point[],
  over: Box | null,
  said: (value: number) => string,
): readonly Height[] => {
  const { least, most } = extent
  const fits = (y: number, lift: string, value: number): readonly Height[] => {
    const box = againstBox(y, lift)
    if (!clearAt(y, points, marks)) return []
    if (over && !apart(box, over)) return []
    return [{ at: against(y, lift), box, text: said(value) }]
  }
  // An extent of no width has one number and nothing else to read, and it is set
  // over the line it names. That line is the foot, which is where a run with no
  // height is drawn.
  if (most === least) {
    return [{ at: against(FOOT, '0'), box: againstBox(FOOT, '0'), text: said(most) }]
  }
  return [...fits(TOP, '-100%', most), ...fits(FOOT, '0', least)]
}

/**
 * Where the knob's own value is set. It rides a line of its own under the
 * picture, so it never prints over a number read off the picture's edges.
 */
export const readingAt = (point: Point | null): CSSProperties => {
  if (!point) return {}
  const back = point.x < LEFT + LABEL ? '0' : point.x > RIGHT - LABEL ? '-100%' : '-50%'
  return { insetInlineStart: `${(point.x / WIDE) * 100}%`, translate: `${back} 0` }
}

/** Where a keystroke takes the knob, and nothing for a keystroke of somebody else's. */
export const walked = (key: string, place: number, places: number): number | null => {
  const last = places - 1
  if (key === 'ArrowLeft' || key === 'ArrowDown') return Math.max(place - 1, 0)
  if (key === 'ArrowRight' || key === 'ArrowUp') return Math.min(place + 1, last)
  if (key === 'Home') return 0
  if (key === 'End') return last
  return null
}

/**
 * The place a pointer stands over, from where it stands across the picture in
 * the picture's own units. A pointer outside the room the line is drawn in
 * stands at the nearer end.
 */
export const placeUnder = (x: number, places: number): number => {
  if (places <= 1) return 0
  const inside = (x - LEFT) / (RIGHT - LEFT)
  return Math.min(Math.max(Math.round(inside * (places - 1)), 0), places - 1)
}

/** Two places past the picture's units, which is as fine as a path needs. */
const round = (value: number): number => Math.round(value * 100) / 100
