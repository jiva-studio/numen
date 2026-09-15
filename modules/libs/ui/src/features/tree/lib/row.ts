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

/** What a row is marked with, and nothing where it is marked with nothing. */
export const getMarkOf = (
  mark: RowMarker | undefined,
  row: RowId | null,
): Record<string, string> => {
  const value = mark?.valueFor(row)
  return mark && value !== null && value !== undefined ? { [mark.attribute]: value } : {}
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

  const walk = (children: readonly Row[], parent: RowId | null, level: number): void => {
    children.forEach((row, at) => {
      const inside = row.rows ?? []
      const opened = row.holds && open.has(row.id)

      shown.push({
        id: row.id,
        name: row.name,
        holds: row.holds,
        parent,
        level,
        last: at === children.length - 1,
        holding: inside.length > 0,
        open: opened,
      })

      if (opened) walk(inside, row.id, level + 1)
    })
  }

  walk(rows, null, 1)
  return shown
}
