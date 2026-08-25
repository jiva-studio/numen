/**
 * What a tree of rows is, as plain values. No DOM, no measurement, no clock.
 *
 * Rows arrive nested, each carrying the rows it holds. Draw order is then the
 * order of one walk down them, and a row's level is the depth it was reached
 * at.
 */

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

/** Where a drag would put what is held. */
export type Landing = { readonly into: RowId } | { readonly before: RowId }

/** The row a landing names. */
const named = (at: Landing): RowId => ('into' in at ? at.into : at.before)

/**
 * What letting go at a height comes to.
 *
 * A row that holds is read in three bands: the middle half means into it, and
 * the quarter at either end means between. Every other row is halved, and each
 * half means between. Past the last row there is nothing to come before, and a
 * landing naming the row being dragged moves nothing; both answer nothing.
 */
export function landing(
  shown: readonly ShownRow[],
  dragging: RowId,
  y: number,
  height: number,
): Landing | null {
  const found = bandAt(shown, y, height)
  return found && named(found) !== dragging ? found : null
}

/** The band a height falls in, as a landing. */
function bandAt(shown: readonly ShownRow[], y: number, height: number): Landing | null {
  if (height <= 0 || y < 0) return null

  const at = Math.floor(y / height)
  const row = shown[at]
  if (!row) return null

  const band = y - at * height
  const edge = row.holds ? height / 4 : height / 2

  if (band < edge) return { before: row.id }
  if (band < height - edge) return { into: row.id }

  const next = shown[at + 1]
  return next ? { before: next.id } : null
}

/** The row a landing puts what is held inside. Null at the top level. */
export const holderOf = (shown: readonly ShownRow[], at: Landing): RowId | null =>
  'into' in at ? at.into : (shown.find((row) => row.id === at.before)?.parent ?? null)

/**
 * A row cannot land in itself, nor in anything it holds. Everything else is
 * allowed, and the top level refuses nothing.
 */
export function refuses(rows: readonly Row[], dragging: RowId, into: RowId | null): boolean {
  if (into === null) return false
  if (into === dragging) return true

  const below = (held: readonly Row[], within: boolean): boolean =>
    held.some(
      (row) => (within && row.id === into) || below(row.rows ?? [], within || row.id === dragging),
    )

  return below(rows, false)
}
