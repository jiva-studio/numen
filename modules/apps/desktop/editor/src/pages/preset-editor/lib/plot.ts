/**
 * Where the one control's curve is drawn, without drawing it.
 *
 * Everything is worked out in the picture's own units and scaled to whatever
 * room the tab has. The curve is the control, so the same arithmetic that puts
 * a place on the line reads a place off a pointer.
 */
import type { Position } from '@numen/ui'
import { costOf } from './curve'
import type { Curve } from '../types'

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

/** One place of the curve, where the picture draws it. */
export type { Position }

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
  const costs = curve.at.map((point) => costOf(curve.goal, point))
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
export const backlogPositionsOf = (
  values: readonly number[],
  extent: Extent,
): readonly Position[] => seriesOf(values, extent, BACKLOG_PLOT)

/**
 * Where every place of the curve is drawn, in the extent the picture is scaled
 * to. The extent is the goal's and not this answer's, so a curve that is flat
 * within it is drawn flat, and an extent of no width lies along the foot.
 */
export const positionsOf = (curve: Curve, extent: Extent): readonly Position[] =>
  seriesOf(
    curve.at.map((point) => costOf(curve.goal, point)),
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
): readonly Position[] => {
  const span = extent.most - extent.least
  // An extent of no width has no height to read: every value in it is the foot
  // of the plot, and the foot is where it is drawn.
  return values.map((value, place) => ({
    x: xOf(place, values.length),
    y:
      span > 0
        ? clamp(plot.foot - (plot.foot - plot.top) * ((value - extent.least) / span), plot)
        : plot.foot,
  }))
}

/** A height inside the plot the line is drawn in. */
const clamp = (y: number, plot: Plot): number => Math.min(Math.max(y, plot.top), plot.foot)

/** The line through those places. */
export const lineOf = (places: readonly Position[]): string =>
  places.map((at, place) => `${place === 0 ? 'M' : 'L'}${round(at.x)} ${round(at.y)}`).join(' ')

/**
 * The stretch the budget the preset keeps does not get through, drawn as a
 * line of its own. A goal that keeps no such account leaves it empty.
 */
export const shortOf = (curve: Curve, places: readonly Position[]): string => {
  const runs: string[] = []
  let run: Position[] = []
  curve.at.forEach((point, place) => {
    const at = places[place]
    if (!at) return
    if (point.canLearnEveryCard) {
      if (run.length > 1) runs.push(lineOf(run))
      run = []
      return
    }
    run.push(at)
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

/** Where a keystroke takes the knob, and nothing for a keystroke of somebody else's. */
export const walkGrid = (key: string, place: number, places: number): number | null => {
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
