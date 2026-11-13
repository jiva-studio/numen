/**
 * The names set over and under a pill: wrapped into rows, given room, and placed.
 * Pure: the width of a title comes in as a number.
 */
import { cutToFit } from './arrange/edges/routing'
import type { PlexName, PlexNames } from '../../types'

/** A name is set at this fraction of the size of a title, and sits back at this opacity until its note is attended to. */
export const NAME_SCALE = 0.75
export const NAME_OPACITY = 0.7

/** The height of a row, in sizes of the type. */
export const ROW_RATIO = 1.3
/** The space between two names of a row, in sizes of the type. */
const SPACING = 0.9
/** The gap between the pill and the nearest row, in rows. */
const PILL_GAP = 0.15

/** A name measured: its title, cut to the width of the node, and the width it takes. */
export interface RowName extends PlexName {
  readonly width: number
}

/** The names that stand on one line, and the width of that line. */
export interface Row {
  readonly names: readonly RowName[]
  readonly width: number
}

/** What the names of one note are measured with. `width` is the widest a row may be: the width of the node. */
export interface NamesRequest {
  readonly fontSize: number
  readonly width: number
  readonly measure: (title: string) => number
}

/** The rows of the names of a note and the room they take over and under its pill. The last row of `above` and the first of `below` are nearest to the pill. */
export interface NamesRoom {
  readonly above: readonly Row[]
  readonly below: readonly Row[]
  readonly rowHeight: number
  readonly spacing: number
  readonly width: number
  readonly top: number
  readonly bottom: number
}

/** A name with the center and size it is drawn at. */
export interface PlacedName extends PlexName {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

function measureName(name: PlexName, request: NamesRequest): RowName {
  const title = cutToFit(name.title, request.width, request.measure)
  return { id: name.id, title, width: Math.min(request.measure(title), request.width) }
}

function wrapNames(names: readonly PlexName[], request: NamesRequest, spacing: number): Row[] {
  const rows: Row[] = []
  let current: RowName[] = []
  let used = 0
  const close = () => {
    if (current.length > 0) rows.push({ names: current, width: used })
    current = []
    used = 0
  }
  for (const name of names) {
    const measured = measureName(name, request)
    if (current.length > 0 && used + spacing + measured.width > request.width) close()
    used = current.length === 0 ? measured.width : used + spacing + measured.width
    current.push(measured)
  }
  close()
  return rows
}

function getHeight(rows: readonly Row[], rowHeight: number): number {
  return rows.length === 0 ? 0 : rows.length * rowHeight + rowHeight * PILL_GAP
}

/** The rows the names wrap into and the room they ask for. A note with no names asks for none. */
export function getNamesRoom(names: PlexNames, request: NamesRequest): NamesRoom {
  const rowHeight = request.fontSize * ROW_RATIO
  const spacing = request.fontSize * SPACING
  const above = wrapNames(names.above, request, spacing)
  const below = wrapNames(names.below, request, spacing)
  return {
    above,
    below,
    rowHeight,
    spacing,
    width: Math.max(0, ...[...above, ...below].map((row) => row.width)),
    top: getHeight(above, rowHeight),
    bottom: getHeight(below, rowHeight),
  }
}

function placeRow(row: Row, room: NamesRoom, at: { x: number; y: number }): PlacedName[] {
  let left = at.x - row.width / 2
  return row.names.map((name) => {
    const placed = {
      id: name.id,
      title: name.title,
      x: left + name.width / 2,
      y: at.y,
      width: name.width,
      height: room.rowHeight,
    }
    left += name.width + room.spacing
    return placed
  })
}

/** The names around a pill, each row centered on it: the first row under the pill nearest to it, the last row over the pill nearest to it. */
export function placeNames(
  room: NamesRoom,
  pill: { readonly x: number; readonly y: number; readonly height: number },
): PlacedName[] {
  const gap = room.rowHeight * PILL_GAP
  const under = room.below.flatMap((row, index) =>
    placeRow(row, room, {
      x: pill.x,
      y: pill.y + pill.height / 2 + gap + (index + 0.5) * room.rowHeight,
    }),
  )
  const over = room.above.flatMap((row, index) =>
    placeRow(row, room, {
      x: pill.x,
      y: pill.y - pill.height / 2 - gap - (room.above.length - index - 0.5) * room.rowHeight,
    }),
  )
  return [...over, ...under]
}
