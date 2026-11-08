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
  readonly hoverRect: {
    readonly left: number
    readonly right: number
    readonly top: number
    readonly bottom: number
  }
  readonly top: number
  readonly height: number
}

export interface RowDragOptions {
  readonly getRows: () => readonly Row[]
  readonly getShownRows: () => readonly ShownRow[]
  readonly measure: () => TreeMetrics | null
  readonly getThreshold: () => number
  readonly getClock: () => Clock
  readonly getCountWords: () => (rows: number) => string
  readonly tell: RowDragTell
}

/** The rows lifted, where they were let go, and that they were let go at all. */
export interface RowDragTell {
  (event: 'drag', rows: readonly RowId[]): void
  (event: 'move', rows: readonly RowId[], target: RowDropTarget): void
  (event: 'drop'): void
}

export interface RowDragState {
  readonly into: ComputedRef<RowId | null>
  readonly before: ComputedRef<RowId | null>
  readonly lifted: ComputedRef<ReadonlySet<RowId>>
  readonly label: ComputedRef<DragLabel | null>
  readonly hasMoved: ComputedRef<boolean>
  readonly at: Readonly<Ref<RowDropTarget | null>>
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
    cancel: () => options.tell('drop'),
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
