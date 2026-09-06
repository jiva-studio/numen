/**
 * What a tree of rows is, as plain values. No DOM, no measurement, no clock.
 *
 * Rows arrive nested, each carrying the rows it holds. Draw order is then the
 * order of one walk down them, and a row's level is the depth it was reached
 * at.
 */

import type { Point } from '../lib/geometry'

/** A row's identity. What it stands for is the caller's to decide. */
export type RowId = string

/**
 * One row: what it is called, and whether it can hold other rows. What is
 * drawn beside the name belongs to whoever renders the tree.
 */
export interface Row {
  readonly id: RowId
  readonly name: string
  readonly holds: boolean
  /** The rows it holds. A row that holds may hold none yet. */
  readonly rows?: readonly Row[]
}

/**
 * An attribute rows are marked with, for something outside the tree to find
 * them by. The tree writes the name and the value it is given and reads
 * neither.
 */
export interface RowMarker {
  readonly attribute: string
  /** What the attribute says on a row, and nothing for a row left unmarked. */
  readonly valueFor: (row: RowId | null) => string | null
}

/** A row in draw order, with everything placing it needs. */
export interface ShownRow {
  readonly id: RowId
  readonly name: string
  readonly holds: boolean
  /** The row holding it. Null at the top level. */
  readonly parent: RowId | null
  /** How deep it stands, counting from one, which is what it is announced as. */
  readonly level: number
  /** The last of the rows its holder holds. */
  readonly last: boolean
  /** Holding at least one row. */
  readonly holding: boolean
  /** What it holds is drawn. */
  readonly open: boolean
}

/**
 * The rows that are drawn, in the order they are drawn. An array, so a
 * viewport is a slice of it.
 */
export function flatten(rows: readonly Row[], open: ReadonlySet<RowId>): readonly ShownRow[] {
  const shown: ShownRow[] = []

  const walk = (held: readonly Row[], parent: RowId | null, level: number): void => {
    held.forEach((row, at) => {
      const inside = row.rows ?? []
      const opened = row.holds && open.has(row.id)

      shown.push({
        id: row.id,
        name: row.name,
        holds: row.holds,
        parent,
        level,
        last: at === held.length - 1,
        holding: inside.length > 0,
        open: opened,
      })

      if (opened) walk(inside, row.id, level + 1)
    })
  }

  walk(rows, null, 1)
  return shown
}

/**
 * What a press means for the selection: a row joining it or leaving it, or the
 * rows from the anchor reaching this one. A press modified by neither makes the
 * row the whole selection.
 */
export interface Press {
  readonly joining: boolean
  readonly reaching: boolean
}

/** A press with nothing held down. */
export const PLAIN: Press = { joining: false, reaching: false }

/** What a press comes to: the selection, and the row a reach is measured from. */
export interface RowSelection {
  readonly rows: readonly RowId[]
  readonly anchor: RowId | null
}

/**
 * The rows from one to another in the order they are drawn, both ends among
 * them, whichever way round the two stand. A range measured from a row that is
 * not drawn is the row it reaches, alone.
 */
export function between(
  shown: readonly ShownRow[],
  from: RowId,
  to: RowId,
): readonly RowId[] {
  const last = shown.findIndex((row) => row.id === to)
  if (last === -1) return []

  const first = shown.findIndex((row) => row.id === from)
  if (first === -1) return [to]

  const [start, end] = first <= last ? [first, last] : [last, first]
  return shown.slice(start, end + 1).map((row) => row.id)
}

/** The rows of a set, in the order they are drawn. */
const inOrder = (shown: readonly ShownRow[], held: ReadonlySet<RowId>): readonly RowId[] =>
  shown.filter((row) => held.has(row.id)).map((row) => row.id)

/**
 * What a press on a row makes the selection.
 *
 * Reaching takes the rows from the anchor to this one and leaves the anchor
 * where it stands; joining takes the row in or out and puts the anchor on it.
 * The rows come back in the order they are drawn, and a row that is not drawn
 * is among none of them.
 */
export function selects(
  shown: readonly ShownRow[],
  selected: readonly RowId[],
  anchor: RowId | null,
  row: RowId,
  press: Press,
): RowSelection {
  if (press.reaching) return { rows: between(shown, anchor ?? row, row), anchor: anchor ?? row }

  if (press.joining) {
    const held = new Set(selected)
    if (held.has(row)) held.delete(row)
    else held.add(row)
    return { rows: inOrder(shown, held), anchor: row }
  }

  return { rows: [row], anchor: row }
}

/** Every row that is drawn, with the anchor left where it stands. */
export const everyRow = (shown: readonly ShownRow[], anchor: RowId | null): RowSelection => ({
  rows: shown.map((row) => row.id),
  anchor: anchor ?? shown[0]?.id ?? null,
})

/**
 * The rows a press on a row drags: the selection, where the row stands in it,
 * and the row alone where it stands outside.
 */
export const dragged = (selected: readonly RowId[], row: RowId): readonly RowId[] =>
  selected.includes(row) ? selected : [row]

/** Whether two selections hold the same rows in the same order. */
export const sameRows = (rows: readonly RowId[], others: readonly RowId[]): boolean =>
  rows.length === others.length && rows.every((row, at) => row === others[at])

/** What is drawn at the pointer while rows are dragged. */
export interface DragLabel {
  /** The name of the one row dragged, or how many there are. */
  readonly says: string
  /** Where the pointer is, which is where it is drawn. */
  readonly at: Point
}

/**
 * What follows the pointer while rows are dragged, and nothing while none are.
 * One row is said by its name; several are said by how many.
 */
export function dragLabel(
  shown: readonly ShownRow[],
  rows: readonly RowId[],
  at: Point,
  counted: (rows: number) => string,
): DragLabel | null {
  const first = rows[0]
  if (first === undefined) return null

  const says =
    rows.length === 1 ? (shown.find((row) => row.id === first)?.name ?? first) : counted(rows.length)
  return { says, at }
}

/** The keys that move the keyboard about a tree, declared once. */
export const TREE_KEYS = ['ArrowDown', 'ArrowUp', 'ArrowRight', 'ArrowLeft', 'Home', 'End'] as const

export type TreeKey = (typeof TREE_KEYS)[number]

export const isTreeKey = (key: string): key is TreeKey =>
  (TREE_KEYS as readonly string[]).includes(key)

/** What a key comes to: a row to turn on the way, and where the keyboard lands. */
export interface Step {
  /** Where the keyboard is afterwards. Null while nothing is drawn. */
  readonly at: RowId | null
  /** The row to open, or to close, before it lands. */
  readonly turn: { readonly row: RowId; readonly open: boolean } | null
}

/**
 * Where a key takes the keyboard.
 *
 * Down and up move a row and stop at the ends. Right opens a closed row and
 * then descends into it; left closes an open row and then climbs to its
 * holder. Home and End go to the ends, and from no row at all every key lands
 * on the first.
 */
export function stepTo(shown: readonly ShownRow[], from: RowId | null, key: TreeKey): Step {
  const first = shown[0]?.id ?? null
  const at = from === null ? -1 : shown.findIndex((row) => row.id === from)
  const here = at === -1 ? undefined : shown[at]
  if (!here) return { at: first, turn: null }

  const stays: Step = { at: here.id, turn: null }

  switch (key) {
    case 'Home':
      return { at: first, turn: null }
    case 'End':
      return { at: shown[shown.length - 1]?.id ?? null, turn: null }
    case 'ArrowDown':
      return { at: shown[at + 1]?.id ?? here.id, turn: null }
    case 'ArrowUp':
      return { at: shown[at - 1]?.id ?? here.id, turn: null }
    case 'ArrowRight':
      if (here.holds && !here.open) return { at: here.id, turn: { row: here.id, open: true } }
      if (here.open && here.holding) return { at: shown[at + 1]?.id ?? here.id, turn: null }
      return stays
    case 'ArrowLeft':
      if (here.open) return { at: here.id, turn: { row: here.id, open: false } }
      return { at: here.parent ?? here.id, turn: null }
  }
}

/**
 * Where held rows would land: inside a row, or above one. Into no row at all
 * is the top level.
 */
export type RowLanding = { readonly into: RowId | null } | { readonly before: RowId }

/** The row a landing names, and nothing for the top level. */
const named = (at: RowLanding): RowId | null => ('into' in at ? at.into : at.before)

/**
 * What letting go at a height comes to.
 *
 * A row that holds is read in three bands: the middle half means into it, and
 * the quarter at either end means between. Every other row is halved, and each
 * half means between. Past the last row is the top level, which is what the
 * tree's own empty area comes to. A landing naming one of the rows being
 * dragged moves nothing and answers nothing.
 */
export function landing(
  shown: readonly ShownRow[],
  dragging: readonly RowId[],
  y: number,
  height: number,
): RowLanding | null {
  const found = bandAt(shown, y, height)
  if (!found) return null

  const on = named(found)
  return on !== null && dragging.includes(on) ? null : found
}

/** The band a height falls in, as a landing. */
function bandAt(shown: readonly ShownRow[], y: number, height: number): RowLanding | null {
  if (height <= 0 || y < 0) return null

  const at = Math.floor(y / height)
  const row = shown[at]
  if (!row) return { into: null }

  const band = y - at * height
  const edge = row.holds ? height / 4 : height / 2

  if (band < edge) return { before: row.id }
  if (band < height - edge) return { into: row.id }

  const next = shown[at + 1]
  return next ? { before: next.id } : { into: null }
}

/** The row a landing puts what is held inside. Null at the top level. */
export const holderOf = (shown: readonly ShownRow[], at: RowLanding): RowId | null =>
  'into' in at ? at.into : (shown.find((row) => row.id === at.before)?.parent ?? null)

/**
 * Rows cannot land in one of themselves, nor in anything one of them holds.
 * Everything else is allowed, and the top level refuses nothing.
 */
export function refuses(
  rows: readonly Row[],
  dragging: readonly RowId[],
  into: RowId | null,
): boolean {
  if (into === null) return false

  const lifted = new Set(dragging)
  if (lifted.has(into)) return true

  const below = (held: readonly Row[], within: boolean): boolean =>
    held.some(
      (row) => (within && row.id === into) || below(row.rows ?? [], within || lifted.has(row.id)),
    )

  return below(rows, false)
}
