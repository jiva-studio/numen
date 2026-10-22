import type { Clock } from '@/shared/lib/clock'
import type { Position } from '@/shared/lib/geometry'
import type { RowDropTarget } from '../lib/drop'
import type { Row, RowId, RowMarker } from '../lib/row'

export interface TreeProps {
  /** What is drawn, nested. */
  rows: readonly Row[]
  /** The rows whose contents are drawn. */
  open?: readonly RowId[]
  /** The rows the selection relies on. */
  selected?: readonly RowId[]
  /** How far the pointer travels before a press becomes a drag. */
  threshold?: number
  /** What the tree is announced as. */
  name?: string
  /** How many rows are being dragged, said at the pointer. */
  counted?: (rows: number) => string
  /**
   * An attribute written onto the rows and onto the tree, for something
   * outside to find them by. Both the name and the values are the caller's,
   * and a row answered with nothing is left unmarked; the tree itself is
   * asked about as no row at all.
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
  /** The selection asked to go. */
  (event: 'remove', rows: readonly RowId[]): void
  /** A menu asked for, and where the pointer was. Nothing for a press off every row. */
  (event: 'menu', row: RowId | null, pos: Position): void
}

export interface TreeSlots {
  /** What is drawn between a row's disclosure and its name. */
  icon(props: { id: RowId; hasChildren: boolean; isOpen: boolean }): unknown
  /** What is said when there is nothing to draw. */
  silence(): unknown
}
