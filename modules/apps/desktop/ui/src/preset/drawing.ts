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

/** The height a curve of one value is drawn at. */
export const MIDDLE = (TOP + FOOT) / 2

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

/** The least and the most a curve costs over its whole grid. */
export const bandOf = (curve: Curve): Band => {
  const costs = curve.at.map((point) => costOf(curve.goal, point))
  if (costs.length === 0) return { least: 0, most: 0 }
  return { least: Math.min(...costs), most: Math.max(...costs) }
}

/**
 * Where every place of the curve is drawn, in the band the picture is scaled
 * to. The band is the goal's and not this answer's, so a curve that is flat
 * within it is drawn flat; a band of no width has no scale and is drawn
 * through the middle.
 */
export const spotsOf = (curve: Curve, band: Band): readonly Spot[] => {
  const span = band.most - band.least
  return curve.at.map((point, place) => ({
    x: xOf(place, curve.at.length),
    y:
      span > 0
        ? held(FOOT - (FOOT - TOP) * ((costOf(curve.goal, point) - band.least) / span))
        : MIDDLE,
  }))
}

/** A height inside the room the line is drawn in. */
const held = (y: number): number => Math.min(Math.max(y, TOP), FOOT)

/** The line through those places. */
export const lineOf = (spots: readonly Spot[]): string =>
  spots.map((spot, at) => `${at === 0 ? 'M' : 'L'}${round(spot.x)} ${round(spot.y)}`).join(' ')

/** The room a number set against a line takes, in the picture's own units. */
export const AXIS_WIDE = 64
export const AXIS_HIGH = 15

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
