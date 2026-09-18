/**
 * Rows lifted off the tree, where letting go of them would put them, and what
 * follows the pointer while they are on their way.
 *
 * Where the pointer is over is worked out from the drawing: the rows are one
 * height each.
 */
import { computed, type ComputedRef, type Ref } from 'vue'
import type { Clock } from '@/shared/lib/clock'
import type { Position } from '@/shared/lib/geometry'
import { usePressDrag } from '@/shared/ui/drag-preview'
import { dragLabel, type DragLabel } from '../lib/drag'
import { holderOf, isRefused, landing, type RowLanding } from '../lib/drop'
import type { Row, RowId, ShownRow } from '../lib/row'

/** The tree as it stands on screen, which only the drawing knows. */
export interface TreeMetrics {
  /** What the tree takes up: a drop lands only over it. */
  readonly over: {
    readonly left: number
    readonly right: number
    readonly top: number
    readonly bottom: number
  }
  /** Where the rows begin, down the window. */
  readonly top: number
  /** How tall one row stands. */
  readonly height: number
}

export interface RowDragOptions {
  /** What is drawn, nested, and the rows of it that are drawn now. */
  readonly getRows: () => readonly Row[]
  readonly getShownRows: () => readonly ShownRow[]
  /** The tree as it stands on screen, and nothing until it is drawn. */
  readonly measure: () => TreeMetrics | null
  /** How far the pointer travels before a press becomes a drag. */
  readonly getThreshold: () => number
  readonly getClock: () => Clock
  /** How many rows are being dragged, said at the pointer. */
  readonly getCountWords: () => (rows: number) => string
  /** What the tree says a drag came to. */
  readonly tell: RowDragTell
}

/** The rows lifted, where they were let go, and that they were let go at all. */
export interface RowDragTell {
  (event: 'drag', rows: readonly RowId[]): void
  (event: 'move', rows: readonly RowId[], at: RowLanding): void
  (event: 'drop'): void
}

export interface RowDragState {
  /** The row a drop would land inside, and the row it would land above. */
  readonly into: ComputedRef<RowId | null>
  readonly before: ComputedRef<RowId | null>
  /** The rows a live drag holds, for asking one row at a time. */
  readonly lifted: ComputedRef<ReadonlySet<RowId>>
  /** What follows the pointer, and nothing until a press has become a drag. */
  readonly label: ComputedRef<DragLabel | null>
  /** Whether the press being made has travelled far enough to be a drag. */
  readonly moved: ComputedRef<boolean>
  /** Where a drop would land, for the tree to mark itself with. */
  readonly at: Readonly<Ref<RowLanding | null>>
  /** A press on a row, which a move turns into a drag. */
  readonly lift: (rows: readonly RowId[], event: PointerEvent) => void
}

export function useRowDrag(options: RowDragOptions): RowDragState {
  const { dragging, at, position, lift } = usePressDrag<readonly RowId[], RowLanding>({
    getThreshold: options.getThreshold,
    getClock: options.getClock,
    getLandingAt,
    settle: (rows, found) => {
      if (found) options.tell('move', rows, found)
      options.tell('drop')
    },
    begin: (rows) => options.tell('drag', rows),
  })

  const into = computed(() => (at.value && 'into' in at.value ? at.value.into : null))
  const before = computed(() => (at.value && 'before' in at.value ? at.value.before : null))

  const lifted = computed(() => new Set(position.value ? (dragging.value?.item ?? []) : []))

  const moved = computed(() => dragging.value?.isMoved === true)

  const label = computed<DragLabel | null>(() => {
    const held = dragging.value
    const where = position.value
    if (!held?.isMoved || !where) return null
    return dragLabel(options.getShownRows(), held.item, where, options.getCountWords())
  })

  /** Where the pointer is, read off the drawing: the rows are one height each. */
  function getLandingAt(rows: readonly RowId[], where: Position): RowLanding | null {
    const drawn = options.measure()
    if (!drawn) return null

    // A pointer that has left the tree is taking what it holds somewhere else.
    const { over } = drawn
    const inside =
      where.x >= over.left && where.x <= over.right && where.y >= over.top && where.y <= over.bottom
    if (!inside) return null

    const shown = options.getShownRows()
    const found = landing(shown, rows, where.y - drawn.top, drawn.height)
    if (!found) return null

    return isRefused(options.getRows(), rows, holderOf(shown, found)) ? null : found
  }

  return { into, before, lifted, label, moved, at, lift }
}
