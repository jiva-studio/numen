import type { Position } from '@numen/ui'
import { PLOT_BOTTOM, PLOT_HEIGHT, PLOT_TOP, getXAtPlace, type Extent, type Plot } from './plot'

const CURVE_PLOT: Plot = { high: PLOT_HEIGHT, top: PLOT_TOP, foot: PLOT_BOTTOM }

/**
 * Where any reading of the curve is drawn: one figure a place, over the same
 * width and against the extent it is scaled to. The line is one such reading
 * and a second series — a backlog under it, say — is another, so a picture that
 * carries two carries them by the same arithmetic.
 */
export const createSeries = (
  values: readonly number[],
  extent: Extent,
  plot: Plot = CURVE_PLOT,
): readonly Position[] => {
  const span = extent.max - extent.min
  // An extent of no width has no height to read: every value in it is the foot
  // of the plot, and the foot is where it is drawn.
  return values.map((value, place) => ({
    x: getXAtPlace(place, values.length),
    y:
      span > 0
        ? clamp(plot.foot - (plot.foot - plot.top) * ((value - extent.min) / span), plot)
        : plot.foot,
  }))
}

const clamp = (y: number, plot: Plot): number => Math.min(Math.max(y, plot.top), plot.foot)
