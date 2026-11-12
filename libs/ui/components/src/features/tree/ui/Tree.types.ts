import type { VNode } from 'vue'
import type { Clock } from '@/shared/lib/time/clock'
import type { Position } from '@/shared/lib/math/geometry'
import type { RowDropTarget } from '../lib/drop'
import type { Row, RowId, RowMarker } from '../lib/row'

/** Props for the tree component. */
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

/** Events emitted by the tree component. */
export interface TreeEvents {
  (event: 'open', row: RowId): void
  (event: 'close', row: RowId): void
  (event: 'select', rows: readonly RowId[]): void
  (event: 'activate', row: RowId): void
  (event: 'rename', row: RowId, name: string): void
  (event: 'move', rows: readonly RowId[], target: RowDropTarget): void
  (event: 'drag', rows: readonly RowId[]): void
  (event: 'drop'): void
  (event: 'remove', rows: readonly RowId[]): void
  (event: 'menu', row: RowId | null, pos: Position): void
}

/** Slots for the tree component. */
export interface TreeSlots {
  icon(props: { id: RowId; hasChildren: boolean; isOpen: boolean }): VNode[]
  silence(): VNode[]
}
