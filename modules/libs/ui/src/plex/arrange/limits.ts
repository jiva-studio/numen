import type { PlexRelatedRole } from '../model'
import { RELATED_ROLES } from '../model'
import { isVertical, type PlexOptions } from './options'

/** How one role wraps: how many along a line, and how many lines. */
export interface RoleLimits {
  readonly perLine: number
  readonly lines: number
}

export type Limits = Readonly<Record<PlexRelatedRole, RoleLimits>>

const clamp = (value: number, low: number, high: number) =>
  Math.max(low, Math.min(high, value))

/** How many boxes of `size` fit end to end in `room`. */
const along = (room: number, size: number, gap: number) =>
  Math.floor((room + gap) / (size + gap))

/**
 * How each role wraps, given the window.
 *
 * What gives is how many go on a line, not how many are shown: a row too wide
 * wraps sooner and runs deeper, into the space above and below. Rows are
 * served first — they are the axis the plex is read on — and a column takes
 * what is left beyond the widest of them.
 */
export function limitsFor(
  options: PlexOptions,
  counts: Readonly<Record<PlexRelatedRole, number>>,
): Limits {
  const asked: RoleLimits = { perLine: options.maxPerLine, lines: options.maxLines }
  const { viewport } = options
  if (!viewport) return everyRole(options, () => asked, () => asked)

  const halfWidth = viewport.width / 2 - options.margin
  const halfHeight = viewport.height / 2 - options.margin
  const { width, height } = options.nodeSize

  const columnRoles = RELATED_ROLES.filter(
    (role) => counts[role] > 0 && !isVertical(options.direction[role]),
  )
  const longestColumn = columnRoles.reduce((most, role) => Math.max(most, counts[role]), 0)
  const hasColumns = longestColumn > 0

  const rowHalf = (perLine: number) => (perLine * width + (perLine - 1) * options.gap) / 2
  const columnsBeside = (perLine: number) =>
    clamp(
      along(halfWidth - rowHalf(perLine) - options.focusGap, width, options.lineGap),
      0,
      options.maxLines,
    )

  // The widest row that still leaves the window able to hold it — and, when
  // there is anything off to the side, room for a column beyond it.
  const row = widestRow(options.maxPerLine, (perLine) => {
    if (rowHalf(perLine) > halfWidth) return false
    return !hasColumns || columnsBeside(perLine) >= 1
  })

  const perColumn = Math.max(1, along(2 * halfHeight, height, options.gap))

  return everyRole(
    options,
    () => ({
      perLine: row,
      lines: clamp(
        along(
          halfHeight - options.focusSize.height / 2 - options.focusGap,
          height,
          options.lineGap,
        ),
        1,
        options.maxLines,
      ),
    }),
    () => ({
      perLine: perColumn,
      lines: hasColumns
        ? Math.max(1, Math.min(columnsBeside(row), Math.ceil(longestColumn / perColumn)))
        : 1,
    }),
  )
}

/** The widest line that fits, or one when none of them do. */
function widestRow(most: number, fits: (perLine: number) => boolean): number {
  for (let perLine = most; perLine >= 1; perLine--) {
    if (fits(perLine)) return perLine
  }
  return 1
}

function everyRole(
  options: PlexOptions,
  forRows: () => RoleLimits,
  forColumns: () => RoleLimits,
): Limits {
  return Object.fromEntries(
    RELATED_ROLES.map((role) => [
      role,
      isVertical(options.direction[role]) ? forRows() : forColumns(),
    ]),
  ) as Limits
}
