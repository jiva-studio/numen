import type { Position } from '@numen/ui'
import type { Curve } from '../../lib/presets'
import { getPointCost } from './curve'
import { BACKLOG_PLOT, type Extent } from './plot'
import { createSeries } from './series'

/**
 * Positions of a backlog. An empty backlog is the foot for every run, so a run
 * with no overdue cards is drawn along the foot and a run with overdue cards is
 * measured upward from it.
 */
export const getBacklogPositions = (
  values: readonly number[],
  extent: Extent,
): readonly Position[] => createSeries(values, extent, BACKLOG_PLOT)

/**
 * Positions of every place of the curve, in the extent the plot is scaled to.
 * The extent is set by the goal and not by this curve, so a curve that is flat
 * within it is drawn flat, and an extent of zero width is drawn along the foot.
 */
export const createPositions = (curve: Curve, extent: Extent): readonly Position[] =>
  createSeries(
    curve.at.map((point) => getPointCost(curve.goal, point)),
    extent,
  )
