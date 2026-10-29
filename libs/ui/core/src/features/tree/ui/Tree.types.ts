import type { Clock } from '@/shared/lib/time/clock'
import type { Position } from '@/shared/lib/math/geometry'
import type { RowDropTarget } from '../lib/drop'
import type { Row, RowId, RowMarker } from '../lib/row'

export interface TreeProps {
  /** What is drawn, nested. */
  rows: readonly Row[]
  /** The rows whose contents are drawn. */
  open?: readonly RowId[]
  /** The rows the selection relies on. */
  selected?: readonly RowId[]
  /** The distance the pointer moves before a press becomes a drag. */
  threshold?: number
  /** What the tree is announced as. */
  name?: string
  /** Returns the label announced at the pointer for the number of rows being dragged. */
  counted?: (rows: number) => string
  /**
   * An attribute written onto the rows and onto the tree, for external code to
   * locate them. Both the name and the values are chosen by the caller. A row
   * mapped to nothing is left unmarked; the tree itself is queried as a null row.
   */
  marking?: RowMarker | undefined
  /** The clock. Browser by default; a test hands in its own. */
  clock?: Clock
}

export interface TreeEvents {
  (event: 'open', row: RowId): void
  (event: 'close', row: RowId): void
  /** The rows the selection now relies on. */
  (event: 'select', rows: readonly RowId[]): void
  /** A row acted on: a double press, or Enter. A row that holds turns as well. */
  (event: 'activate', row: RowId): void
  /** A name typed and committed. */
  (event: 'rename', row: RowId, name: string): void
  /** The rows let go somewhere, all of them drop target in the one place. */
  (event: 'move', rows: readonly RowId[], target: RowDropTarget): void
  /**
   * The rows lifted clear of the tree, on their way across whatever is drawn
   * beside it. Where they end up there is not the tree's to say.
   */
  (event: 'drag', rows: readonly RowId[]): void
  /** The rows release, wherever the pointer had got to. */
  (event: 'drop'): void
  /** The selection was removed. */
  (event: 'remove', rows: readonly RowId[]): void
  /** A menu was requested at a position. The row is null for a press outside every row. */
  (event: 'menu', row: RowId | null, pos: Position): void
}

export interface TreeSlots {
  /** What is drawn between a row's disclosure and its name. */
  icon(props: { id: RowId; hasChildren: boolean; isOpen: boolean }): unknown
  /** The content shown when there is nothing to draw. */
  silence(): unknown
}
