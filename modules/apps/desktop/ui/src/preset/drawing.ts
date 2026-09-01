/**
 * Where the one control's curve is drawn, without drawing it.
 *
 * Everything is worked out in the picture's own units and scaled to whatever
 * room the tab has. The curve is the control, so the same arithmetic that puts
 * a point on the line reads a place off a pointer.
 */
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
export interface Room {
  readonly high: number
  readonly top: number
  readonly foot: number
}

/** The curve's own room, which is the picture. */
export const PLOT: Room = { high: HIGH, top: TOP, foot: FOOT }

/** The band's room under it, which is short and shares the picture's width. */
export const BAND_HIGH = 76
export const BAND: Room = { high: BAND_HIGH, top: 12, foot: BAND_HIGH - 12 }

/** The faint lines the picture is banded by, as shares of the room's height. */
export const BANDS: readonly number[] = [0.25, 0.5, 0.75]

/** How far the name of a mark stands above it, and how wide it is set. */
export const LIFT = 9
export const LABEL = 34

/** One place of the curve, where the picture draws it. */
export interface Spot {
  readonly x: number
  readonly y: number
}

/** Where one place of a grid stands across the picture. */
export const xOf = (place: number, places: number): number =>
  places <= 1 ? LEFT : LEFT + ((RIGHT - LEFT) * place) / (places - 1)

/** The height of one of the faint lines. */
export const yOfBand = (share: number): number => FOOT - (FOOT - TOP) * share

/** The stretch of cost the picture is scaled to. */
export interface Band {
  readonly least: number
  readonly most: number
}

/**
 * The band a curve is drawn against: nothing at the foot, and the most it ever
 * costs over the top. A count does not go below nothing, so the foot of the
 * picture is nothing under every run and a height is read against it.
 */
export const bandOf = (curve: Curve): Band => {
  const costs = curve.at.map((point) => costOf(curve.goal, point))
  return { least: 0, most: costs.length === 0 ? 0 : Math.max(0, ...costs) }
}

/**
 * The band a backlog is drawn against: nothing overdue at the foot, and the
 * most this pace ever stands at over the top. It is scaled to the one place
 * the knob stands at, so a day too short to carry what falls due is drawn
 * climbing over the height it has.
 */
export const bandOfBacklog = (backlog: readonly number[]): Band => ({
  least: 0,
  most: backlog.length === 0 ? 0 : Math.max(...backlog),
})

/**
 * Where a backlog is drawn. Nothing overdue is the foot under every run, so a
 * run holding nothing at all lies along the floor and a run holding anything
 * is measured up from it.
 */
export const backlogSpotsOf = (values: readonly number[], band: Band): readonly Spot[] =>
  seriesOf(values, band, BAND)

/**
 * Where every place of the curve is drawn, in the band the picture is scaled
 * to. The band is the goal's and not this answer's, so a curve that is flat
 * within it is drawn flat, and a band of no width lies along the foot.
 */
export const spotsOf = (curve: Curve, band: Band): readonly Spot[] =>
  seriesOf(
    curve.at.map((point) => costOf(curve.goal, point)),
    band,
  )

/**
 * Where any reading of the curve is drawn: one figure a place, over the same
 * width and against the band it is scaled to. The line is one such reading and
 * a second series — a backlog under it, say — is another, so a picture that
 * carries two carries them by the same arithmetic.
 */
export const seriesOf = (
  values: readonly number[],
  band: Band,
  room: Room = PLOT,
): readonly Spot[] => {
  const span = band.most - band.least
  // A band of no width has no height to read: every value in it is the foot of
  // the band, and the foot is where it is drawn.
  return values.map((value, place) => ({
    x: xOf(place, values.length),
    y:
      span > 0
        ? held(room.foot - (room.foot - room.top) * ((value - band.least) / span), room)
        : room.foot,
  }))
}

/** A height inside the room the line is drawn in. */
const held = (y: number, room: Room): number => Math.min(Math.max(y, room.top), room.foot)

/** The line through those places. */
export const lineOf = (spots: readonly Spot[]): string =>
  spots.map((spot, at) => `${at === 0 ? 'M' : 'L'}${round(spot.x)} ${round(spot.y)}`).join(' ')

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
export const PERCH_WIDE = 104
export const PERCH_HIGH = 48
export const PERCH_GAP = 24

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
  spots: readonly Spot[],
  marks: readonly (Spot | null)[],
): boolean => {
  const near = (spot: Spot): boolean =>
    spot.x <= LEFT + AXIS_WIDE && Math.abs(spot.y - y) <= AXIS_HIGH
  if (marks.some((spot) => spot !== null && near(spot))) return false
  return !spots.some(near)
}

/**
 * The stretch the budget the preset keeps does not get through, drawn as a
 * line of its own. A goal that keeps no such account leaves it empty.
 */
export const shortOf = (curve: Curve, spots: readonly Spot[]): string => {
  const runs: string[] = []
  let run: Spot[] = []
  curve.at.forEach((point, place) => {
    const spot = spots[place]
    if (!spot) return
    if (point.enough) {
      if (run.length > 1) runs.push(lineOf(run))
      run = []
      return
    }
    run.push(spot)
  })
  if (run.length > 1) runs.push(lineOf(run))
  return runs.join(' ')
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
