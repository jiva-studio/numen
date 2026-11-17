/**
 * Geometry of the curve control's plot, without rendering it.
 *
 * All geometry is in plot units and scaled to the available tab space. The
 * curve is the control, so the arithmetic that maps a place to a position on
 * the line also maps a pointer position to a place.
 */
import type { Position } from '@numen/ui'
import { getPointCost } from './curve'
import type { Curve } from '../../lib/presets'

/** Plot units. The plot is scaled uniformly, so a circle stays a circle. */
export const PLOT_WIDTH = 560
export const PLOT_HEIGHT = 190

/** Area the line is drawn in, inside the plot. */
export const PLOT_LEFT = 12
export const PLOT_RIGHT = PLOT_WIDTH - 12
export const PLOT_TOP = 16
export const PLOT_BOTTOM = PLOT_HEIGHT - 16

export interface Plot {
  readonly high: number
  readonly top: number
  readonly foot: number
}

/** The backlog plot below the curve plot: shorter, and the same width. */
export const BACKLOG_PLOT_HEIGHT = 76
export const BACKLOG_PLOT: Plot = {
  high: BACKLOG_PLOT_HEIGHT,
  top: 12,
  foot: BACKLOG_PLOT_HEIGHT - 12,
}

/** The faint gridlines across the plot, as shares of the plot height. */
export const GRIDLINES: readonly number[] = [0.25, 0.5, 0.75]

/** One place of the curve, as a position on the plot. */
export type { Position }

/** Horizontal position of one place of the grid on the plot. */
export const getXAtPlace = (place: number, places: number): number =>
  places <= 1 ? PLOT_LEFT : PLOT_LEFT + ((PLOT_RIGHT - PLOT_LEFT) * place) / (places - 1)

/** The height of one of the faint lines. */
export const yOfGridline = (share: number): number => PLOT_BOTTOM - (PLOT_BOTTOM - PLOT_TOP) * share

/** The range of cost the plot is scaled to. */
export interface Extent {
  readonly min: number
  readonly max: number
}

/**
 * The extent a curve is drawn against: zero at the foot, and the highest cost
 * of the curve at the top. A count is never negative, so the foot of the plot
 * is zero for every run and a height is read against it.
 */
export const getExtent = (curve: Curve): Extent => {
  const costs = curve.at.map((point) => getPointCost(curve.goal, point))
  return { min: 0, max: Math.max(0, ...costs) }
}

/**
 * The extent a backlog is drawn against: nothing overdue at the foot, and the
 * climbing over the height it has.
 */
export const getBacklogExtent = (backlog: readonly number[]): Extent => ({
  min: 0,
  max: backlog.length === 0 ? 0 : Math.max(...backlog),
})

/** The line through those places. */
export const createLinePath = (places: readonly Position[]): string =>
  places
    .map((point, place) => `${place === 0 ? 'M' : 'L'}${round(point.x)} ${round(point.y)}`)
    .join(' ')

/**
 * The stretch the budget the preset keeps does not get through, drawn as a
 * line of its own. A goal that keeps no such account leaves it empty.
 */
export const getShortPath = (curve: Curve, places: readonly Position[]): string => {
  const runs: string[] = []
  let run: Position[] = []
  curve.at.forEach((point, place) => {
    const pos = places[place]
    if (!pos) return
    if (point.canLearnEveryCard) {
      if (run.length > 1) runs.push(createLinePath(run))
      run = []
      return
    }
    run.push(pos)
  })
  if (run.length > 1) runs.push(createLinePath(run))
  return runs.join(' ')
}

const getDayLimit = (curve: Curve, place: number): number =>
  curve.goal === 'date' ? Math.max(Math.round(curve.grid[place] ?? 0), 0) : -1

export const getBacklogRun = (curve: Curve, place: number): readonly number[] => {
  const run = curve.at[place]?.backlog ?? []
  const days = getDayLimit(curve, place)
  return days < 0 ? run : run.slice(0, days)
}

/** The place a keystroke moves the knob to, or null for a key that does not move it. */
export const getKeyPlace = (key: string, place: number, places: number): number | null => {
  const last = places - 1
  if (key === 'ArrowLeft' || key === 'ArrowDown') return Math.max(place - 1, 0)
  if (key === 'ArrowRight' || key === 'ArrowUp') return Math.min(place + 1, last)
  if (key === 'Home') return 0
  if (key === 'End') return last
  return null
}

/**
 * The place of the curve at a horizontal position, in plot units. A pointer
 * outside the area the line is drawn in maps to the first or last place.
 */
export const getPlaceAtX = (x: number, places: number): number => {
  const inside = (x - PLOT_LEFT) / (PLOT_RIGHT - PLOT_LEFT)
  return Math.max(Math.min(Math.round(inside * (places - 1)), places - 1), 0)
}

const round = (value: number): number => Math.round(value * 100) / 100
