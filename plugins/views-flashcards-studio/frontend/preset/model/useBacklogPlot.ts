import { computed } from 'vue'
import { createBacklogAxisLabels } from '../lib/label'
import { getBacklogExtent, getBacklogPositions, getBacklogRun, createLinePath } from '../lib/plot'
import type { Extent } from '../lib/plot'
import { WORDS as words } from '../lib/words'
import type { Curve } from '../../lib/presets'

export interface BacklogPlotSource {
  readonly curve: Curve
  readonly place: number
  readonly isHonest: boolean
}

/** The backlog at the knob position as a line, with the labels of its axes. */
export function useBacklogPlot(source: BacklogPlotSource) {
  /** The backlog at the knob position, one figure per day. */
  const backlog = computed<readonly number[]>(() => getBacklogRun(source.curve, source.place))

  /**
   * The extent it is drawn against, which is the most any place of the curve ever
   * moves, and a place whose run is cut shorter than another's is drawn against
   * the same height as the rest.
   */
  const extent = computed<Extent>(() =>
    getBacklogExtent(
      source.curve.at.flatMap((_, place) => [...getBacklogRun(source.curve, place)]),
    ),
  )

  const positions = computed(() => getBacklogPositions(backlog.value, extent.value))
  const line = computed(() => createLinePath(positions.value))

  /** Whether there is a backlog to draw at all. */
  const hasBacklog = computed(() => source.isHonest && backlog.value.length > 1)

  /**
   * The ends of the extent, with the lines they are the height of. Nothing
   * overdue is the base, so a run with no backlog is that one number on the
   * baseline.
   */
  const heights = computed(() =>
    createBacklogAxisLabels(extent.value, positions.value, (value) =>
      words.formatBacklogCards(value),
    ),
  )

  /** The days at either end of the extent, which the grid does not label. */
  const ends = computed(() => [
    words.formatBacklogDays(1),
    words.formatBacklogDays(backlog.value.length),
  ])

  return { line, hasBacklog, heights, ends }
}
