import { RELATED_SEATS, type PlexRelatedSeat } from '../seat'
import { isVertical, type PlexOptions } from './options'

/** How one seat wraps: how many along a line, and how many lines. */
export interface RoleLimits {
  readonly perLine: number
  readonly lines: number
}

export type Limits = Readonly<Record<PlexRelatedSeat, RoleLimits>>

const clamp = (value: number, low: number, high: number) => Math.max(low, Math.min(high, value))

/** How many boxes of `size` fit end to end in `room`. */
const countAlong = (room: number, size: number, gap: number) =>
  Math.floor((room + gap) / (size + gap))

/**
 * How each seat wraps, given the window.
 *
 * What gives is how many go on a line, not how many are shown: a row too wide
 * wraps sooner and runs deeper, into the space above and below. Rows are
 * served first — they are the axis the plex is read on — and a column takes
 * what is left beyond the widest of them.
 */
export function limitsFor(
  options: PlexOptions,
  counts: Readonly<Record<PlexRelatedSeat, number>>,
): Limits {
  const asked: RoleLimits = { perLine: options.maxPerLine, lines: options.maxLines }
  const { viewport } = options
  if (!viewport)
    return everySeat(
      options,
      () => asked,
      () => asked,
    )

  const halfWidth = viewport.width / 2 - options.margin
  const halfHeight = viewport.height / 2 - options.margin
  const { width, height } = options.nodeSize

  const columnSeats = RELATED_SEATS.filter(
    (seat) => counts[seat] > 0 && !isVertical(options.direction[seat]),
  )
  const longestColumn = columnSeats.reduce((most, seat) => Math.max(most, counts[seat]), 0)
  const hasColumns = longestColumn > 0

  // What a column has to clear, which is the row *or the focus*, whichever
  // reaches further. A row of one is narrower than the focus it sits above, and
  // the arrangement clears the wider of the two — so modelling the row alone
  // promises room that is not there and the column is drawn past the edge.
  const rowHalf = (perLine: number) =>
    Math.max(options.focusSize.width / 2, (perLine * width + (perLine - 1) * options.gap) / 2)
  /** How many lines of a column stand beyond something of this half-width. */
  const columnsBeyond = (clear: number) =>
    clamp(
      countAlong(halfWidth - clear - options.focusGap, width, options.lineGap),
      0,
      options.maxLines,
    )

  const columnsBeside = (perLine: number) => columnsBeyond(rowHalf(perLine))

  const perColumn = Math.max(1, countAlong(2 * halfHeight, height, options.gap))

  // How far a column reaches from the axis, against where the nearest row
  // begins. A column is centred on the focus, so a short one stands level with
  // the focus and nothing else, and the rows above and below have the width of
  // the window to themselves.
  const columnReach =
    ((Math.min(longestColumn, perColumn) - 1) / 2) * (height + options.gap) + height / 2
  const rowsBegin = options.focusSize.height / 2 + options.focusGap
  const columnsMeetRows = hasColumns && columnReach > rowsBegin

  // What a column has to clear: the row where the two meet, the focus alone
  // where they do not.
  const columnClears = (perLine: number) =>
    columnsMeetRows ? rowHalf(perLine) : options.focusSize.width / 2

  // Whether the window has any row width at all that seats a column beside it.
  // A narrower row leaves more room, so the narrowest is the one that answers.
  const seatsColumn = columnsMeetRows && columnsBeside(1) >= 1

  // The widest row that still leaves the window able to hold it — and, where a
  // column can be seated, room for one beyond it. It is measured from what the
  // window holds at the settings, which are the closest the gaps ever pack.
  const row = findWidestRow(countAlong(2 * halfWidth, width, options.gap), (perLine) => {
    if (rowHalf(perLine) > halfWidth) return false
    return !seatsColumn || columnsBeside(perLine) >= 1
  })

  return everySeat(
    options,
    () => ({
      perLine: row,
      lines: clamp(
        countAlong(
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
      // No lines at all when nothing fits beside the rows. What cannot be
      // drawn is reported as overflow, which is what the reader can act on;
      // drawing it off the edge of the window is not.
      lines: hasColumns
        ? Math.min(columnsBeyond(columnClears(row)), Math.ceil(longestColumn / perColumn))
        : 1,
    }),
  )
}

/** The widest line that fits, or one when none of them do. */
function findWidestRow(most: number, fits: (perLine: number) => boolean): number {
  for (let perLine = most; perLine >= 1; perLine--) {
    if (fits(perLine)) return perLine
  }
  return 1
}

function everySeat(
  options: PlexOptions,
  forRows: () => RoleLimits,
  forColumns: () => RoleLimits,
): Limits {
  return Object.fromEntries(
    RELATED_SEATS.map((seat) => [
      seat,
      isVertical(options.direction[seat]) ? forRows() : forColumns(),
    ]),
  ) as Limits
}
