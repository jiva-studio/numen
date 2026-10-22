import { clamp } from '@/shared/lib/math/bounds'
import { RELATION_NAMES, type PlexRelation } from '../../relation'
import { isVertical, type PlexOptions } from '../options'

/** How one relation wraps: how many along a line, and how many lines. */
export interface RoleLimits {
  readonly perLine: number
  readonly lines: number
}

export type Limits = Readonly<Record<PlexRelation, RoleLimits>>

/** How many boxes of `size` fit end to end in `room`. */
const countAlong = (room: number, size: number, gap: number) =>
  Math.floor((room + gap) / (size + gap))

/**
 * How each relation wraps, given the window.
 *
 * What gives is how many go on a line, not how many are shown: a row too wide
 * wraps sooner and runs deeper, into the space above and below. Rows are
 * served first — they are the axis the plex is read on — and a column takes
 * what is left beyond the widest of them.
 */
export function getLimits(
  options: PlexOptions,
  counts: Readonly<Record<PlexRelation, number>>,
): Limits {
  const roleLimits: RoleLimits = { perLine: options.maxPerLine, lines: options.maxLines }
  const { viewport } = options
  if (!viewport)
    return getRelationLimits(
      options,
      () => roleLimits,
      () => roleLimits,
    )

  return getViewportLimits(options, counts, viewport)
}

function getViewportLimits(
  options: PlexOptions,
  counts: Readonly<Record<PlexRelation, number>>,
  viewport: { readonly width: number; readonly height: number },
): Limits {
  const halfWidth = viewport.width / 2 - options.margin
  const halfHeight = viewport.height / 2 - options.margin
  const { width, height } = options.nodeSize

  const columnRelations = RELATION_NAMES.filter(
    (relation) => !isVertical(options.direction[relation]),
  )
  const longestColumn = columnRelations.reduce(
    (most, relation) => Math.max(most, counts[relation]),
    0,
  )
  const hasColumns = longestColumn > 0

  const rowHalf = (perLine: number) =>
    Math.max(options.focusSize.width / 2, (perLine * width + (perLine - 1) * options.gap) / 2)
  const columnsBeyond = (clear: number) =>
    clamp(countAlong(halfWidth - clear - options.focusGap, width, options.lineGap), {
      min: 0,
      max: options.maxLines,
    })

  const columnsBeside = (perLine: number) => columnsBeyond(rowHalf(perLine))
  const perColumn = Math.max(1, countAlong(2 * halfHeight, height, options.gap))

  const columnReach =
    ((Math.min(longestColumn, perColumn) - 1) / 2) * (height + options.gap) + height / 2
  const rowsBegin = options.focusSize.height / 2 + options.focusGap
  const columnsMeetRows = hasColumns && columnReach > rowsBegin

  const getColumnClearance = (perLine: number) =>
    columnsMeetRows ? rowHalf(perLine) : options.focusSize.width / 2

  const canFitColumn = columnsMeetRows && columnsBeside(1) >= 1

  const row = getWidestRow(countAlong(2 * halfWidth, width, options.gap), (perLine) => {
    if (rowHalf(perLine) > halfWidth) return false
    return !canFitColumn || columnsBeside(perLine) >= 1
  })

  return getRelationLimits(
    options,
    () => ({
      perLine: row,
      lines: clamp(
        countAlong(
          halfHeight - options.focusSize.height / 2 - options.focusGap,
          height,
          options.lineGap,
        ),
        { min: 1, max: options.maxLines },
      ),
    }),
    () => ({
      perLine: perColumn,
      lines: hasColumns
        ? Math.min(columnsBeyond(getColumnClearance(row)), Math.ceil(longestColumn / perColumn))
        : 1,
    }),
  )
}

/** The widest line that fits, or one when none of them do. */
function getWidestRow(most: number, fits: (perLine: number) => boolean): number {
  for (let perLine = most; perLine > 0; perLine--) {
    if (fits(perLine)) return perLine
  }
  return 1
}

function getRelationLimits(
  options: PlexOptions,
  forRows: () => RoleLimits,
  forColumns: () => RoleLimits,
): Limits {
  return Object.fromEntries(
    RELATION_NAMES.map((relation) => [
      relation,
      isVertical(options.direction[relation]) ? forRows() : forColumns(),
    ]),
  ) as Limits
}
