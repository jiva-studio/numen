import { computed, type Ref } from 'vue'
import { createRefRegistry } from '@/shared/model/refs'
import { measureElementRect } from '@/shared/api/dom'
import type { Clock } from '@/shared/lib/time/clock'
import { flatten, type Row, type RowId } from '../lib/row'
import { useRowDrag, type RowDragEmit } from './drag'
import { createTreeGestures, createTreeRenaming, type TreeGesturesEmit } from './gestures'
import { useTreeLayout } from './layout'
import { useDrawnRows } from './rows'
import { useRowSelection } from './selection'

/** What a tree reads from its props, read again on every use. */
export interface TreeInput {
  readonly rows: readonly Row[]
  readonly open: readonly RowId[]
  readonly selected: readonly RowId[]
  readonly threshold: number
  readonly clock: Clock
  readonly counted: (rows: number) => string
}

export interface TreeOptions {
  readonly input: TreeInput
  readonly box: Ref<HTMLElement | null>
  readonly list: Ref<HTMLElement | null>
  readonly renamingPath: Ref<RowId | null>
  readonly select: (rows: readonly RowId[]) => void
  readonly emit: TreeGesturesEmit & RowDragEmit
}

/** The visible rows of a tree, their layout, selection, drag, gestures and renaming. */
export function useTree(options: TreeOptions) {
  const { input, box, list, renamingPath, select, emit } = options

  const visibleRows = computed(() => flatten(input.rows, new Set(input.open)))

  function getRowHeight(): number {
    const listEl = list.value
    const anyRow = listEl?.querySelector('[data-tree-row]')
    const height = anyRow ? measureElementRect(anyRow).height : undefined
    return height && height > 0 ? height : 24
  }

  const layout = useTreeLayout({ box, list, visibleRows, getRowHeight })

  const rows = useDrawnRows(
    () => visibleRows.value,
    () => input.selected,
    (index) => layout.virtual.scrollToRow(index),
  )
  const { getRef: getRowRef } = createRefRegistry<RowId>(rows.setRowElement)

  const selection = useRowSelection(
    () => visibleRows.value,
    () => input.selected,
    select,
  )

  const drag = useRowDrag({
    getRows: () => input.rows,
    getShownRows: () => visibleRows.value,
    measure: () => {
      const listEl = list.value
      const overBox = box.value ? measureElementRect(box.value) : undefined
      if (!listEl || !overBox) return null
      const anyRow = listEl.querySelector('[data-tree-row]')
      const rowHeight = anyRow ? measureElementRect(anyRow).height : 24
      return { hoverRect: overBox, top: measureElementRect(listEl).top, height: rowHeight }
    },
    getThreshold: () => input.threshold,
    getClock: () => input.clock,
    getCountWords: () => input.counted,
    emit,
  })

  const gestures = createTreeGestures({
    getShownRows: () => visibleRows.value,
    getSelected: () => input.selected,
    rows,
    selection,
    drag,
    getMenuAt: (row) => {
      const rowEl = rows.getRowElement(row)
      const targetBox = rowEl ? measureElementRect(rowEl) : undefined
      return targetBox ? { x: targetBox.left, y: targetBox.bottom } : null
    },
    emit,
  })

  const renaming = createTreeRenaming(renamingPath, rows, emit)

  return {
    visibleRows,
    layout,
    rows,
    getRowRef,
    picked: selection.picked,
    drag,
    gestures,
    renaming,
  }
}
