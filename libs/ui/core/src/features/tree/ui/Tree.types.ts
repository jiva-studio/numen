import type { Clock } from '@/shared/lib/time/clock'
import type { Position } from '@/shared/lib/math/geometry'
import type { RowDropTarget } from '../lib/drop'
import type { Row, RowId, RowMarker } from '../lib/row'

export interface TreeProps {
  rows: readonly Row[]
  open?: readonly RowId[]
  selected?: readonly RowId[]
  threshold?: number
  name?: string
  counted?: (rows: number) => string
  marking?: RowMarker | undefined
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
