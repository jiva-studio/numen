/**
 * Rows lifted off the tree, where letting go of them would put them, and what
 * follows the pointer while they are on their way.
 *
 * Where the pointer is over is worked out from the drawing: the rows are one
 * height each.
 */
import { computed, type ComputedRef, type Ref } from 'vue'
import type { Clock } from '@/shared/lib/time/clock'
import type { Position } from '@/shared/lib/math/geometry'
import { usePressDrag } from '@/shared/ui/drag-preview'
import { dragLabel, type DragLabel } from '../lib/drag'
import { findHolder, isRefused, getRowDropTarget, type RowDropTarget } from '../lib/drop'
import type { Row, RowId, ShownRow } from '../lib/row'

/** Screen measurements of the tree, available only from the rendering. */
export interface TreeMetrics {
  /** What the tree takes up: a drop lands only over it. */
  readonly hoverRect: {
    readonly left: number
    readonly right: number
    readonly top: number
    readonly bottom: number
  }
  /** Where the rows begin, down the window. */
  readonly top: number
  /** The height of one row. */
  readonly height: number
}

export interface RowDragOptions {
  /** What is drawn, nested, and the rows of it that are drawn now. */
  readonly getRows: () => readonly Row[]
  readonly getShownRows: () => readonly ShownRow[]
  /** The tree as laid out on screen, and null until it is drawn. */
  readonly measure: () => TreeMetrics | null
  /** The distance the pointer moves before a press becomes a drag. */
  readonly getThreshold: () => number
  readonly getClock: () => Clock
  /** The number of rows being dragged, shown at the pointer. */
  readonly getCountWords: () => (rows: number) => string
  /** The callback through which the tree reports drag events. */
  readonly tell: RowDragTell
}

/** The rows lifted, where they were let go, and that they were let go at all. */
export interface RowDragTell {
  (event: 'drag', rows: readonly RowId[]): void
  (event: 'move', rows: readonly RowId[], target: RowDropTarget): void
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
  readonly hasMoved: ComputedRef<boolean>
  /** Where a drop would land, for the tree to mark itself with. */
  readonly at: Readonly<Ref<RowDropTarget | null>>
  /** A press on a row, which a move turns into a drag. */
  readonly lift: (rows: readonly RowId[], event: PointerEvent) => void
}

export function useRowDrag(options: RowDragOptions): RowDragState {
  const {
    dragging,
    at: dropTarget,
    position,
    lift,
  } = usePressDrag<readonly RowId[], RowDropTarget>({
    getThreshold: options.getThreshold,
    getClock: options.getClock,
    getDropTargetAt,
    commit: (rows, found) => {
      if (found) options.tell('move', rows, found)
      options.tell('drop')
    },
    begin: (rows) => options.tell('drag', rows),
  })

  const into = computed(() =>
    dropTarget.value && 'into' in dropTarget.value ? dropTarget.value.into : null,
  )
  const before = computed(() =>
    dropTarget.value && 'before' in dropTarget.value ? dropTarget.value.before : null,
  )

  const lifted = computed(() => new Set(position.value ? dragging.value!.item : []))

  const hasMoved = computed(() => dragging.value?.hasMoved === true)

  const label = computed<DragLabel | null>(() => {
    const currentDrag = dragging.value
    const where = position.value
    if (!currentDrag?.hasMoved || !where) return null
    return dragLabel(options.getShownRows(), currentDrag.item, {
      at: where,
      getCountWords: options.getCountWords(),
    })
  })

  /** Where the pointer is, read off the drawing: the rows are one height each. */
  function getDropTargetAt(rows: readonly RowId[], where: Position): RowDropTarget | null {
    const measurements = options.measure()
    if (!measurements) return null

    // A pointer that has left the tree is taking what it holds somewhere else.
    const { hoverRect } = measurements
    const inside =
      where.x >= hoverRect.left &&
      where.x <= hoverRect.right &&
      where.y >= hoverRect.top &&
      where.y <= hoverRect.bottom
    if (!inside) return null

    const shownRows = options.getShownRows()
    const found = getRowDropTarget(shownRows, rows, {
      y: where.y - measurements.top,
      height: measurements.height,
    })
    if (!found) return null

    return isRefused(options.getRows(), rows, findHolder(shownRows, found)) ? null : found
  }

  return { into, before, lifted, label, hasMoved, at: dropTarget, lift }
}
