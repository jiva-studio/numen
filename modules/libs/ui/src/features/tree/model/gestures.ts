/**
 * What a press, a double press, a key or a menu asked for on a row evaluates to.
 *
 * The selection and the drag work out what a gesture makes of them; this is
 * where a gesture is read and the tree says what happened.
 */
import type { Ref } from 'vue'
import { getDraggedRows } from '../lib/drag'
import type { RowId, ShownRow } from '../lib/row'
import { PLAIN, type Press } from '../lib/select'
import { isTreeKey, stepTo, type Step } from '../lib/step'
import type { Position } from '@/shared/lib/geometry'
import type { RowDragState } from './drag'
import type { DrawnRowsState } from './rows'
import type { RowSelectionState } from './selection'

/** What the tree says a gesture came to. */
export interface TreeGesturesTell {
  (event: 'open', row: RowId): void
  (event: 'close', row: RowId): void
  (event: 'activate', row: RowId): void
  (event: 'rename', row: RowId, name: string): void
  (event: 'remove', rows: readonly RowId[]): void
  (event: 'menu', row: RowId | null, at: Position): void
}

export interface TreeGesturesOptions {
  readonly getShownRows: () => readonly ShownRow[]
  readonly getSelected: () => readonly RowId[]
  /** The row whose name is in a field. */
  readonly renamingPath: Ref<RowId | null>
  readonly rows: DrawnRowsState
  readonly selection: RowSelectionState
  readonly drag: RowDragState
  /** Where a menu asked for by the keyboard opens, off the row it is on. */
  readonly getMenuAt: (row: RowId) => Position | null
  readonly tell: TreeGesturesTell
}

export interface TreeGesturesState {
  readonly onContextMenu: (event: MouseEvent) => void
  readonly onRowContextMenu: (row: ShownRow, event: MouseEvent) => void
  readonly onRowFocus: (row: RowId) => void
  readonly onRowPointerDown: (row: RowId, event: PointerEvent) => void
  readonly onRowClick: (row: ShownRow) => void
  readonly onRowDoubleClick: (row: ShownRow) => void
  readonly onRename: (row: RowId, name: string) => void
  readonly onAbandon: (row: RowId) => void
  readonly onFieldBlur: () => void
  readonly onKeyDown: (event: KeyboardEvent) => void
}

function useTreeRenaming(
  renamingPath: Ref<RowId | null>,
  rows: DrawnRowsState,
  tell: TreeGesturesTell,
) {
  function onRename(row: RowId, name: string): void {
    renamingPath.value = null
    tell('rename', row, name)
    void rows.focusRow(row)
  }

  function onAbandon(row: RowId): void {
    renamingPath.value = null
    void rows.focusRow(row)
  }

  function onFieldBlur(): void {
    renamingPath.value = null
  }

  return { onRename, onAbandon, onFieldBlur }
}

interface TreeKeyboardHelpers {
  readonly activateRow: (row: ShownRow) => void
  readonly requestMenu: (row: ShownRow | null, at: Position) => void
}

function useTreeKeyboard(options: TreeGesturesOptions, helpers: TreeKeyboardHelpers) {
  const { getShownRows, getSelected, rows, selection, tell } = options

  function takeSelectionKey(event: KeyboardEvent): void {
    const chorded = event.ctrlKey || event.metaKey
    if (chorded && event.key.toLowerCase() === 'a') {
      event.preventDefault()
      selection.selectEveryRow()
    } else if (event.key === 'Delete' || event.key === 'Backspace') {
      event.preventDefault()
      if (getSelected().length > 0) tell('remove', getSelected())
    }
  }

  function takeRowKey(event: KeyboardEvent, on: ShownRow): void {
    if (event.key === 'Enter') {
      event.preventDefault()
      helpers.activateRow(on)
    } else if (event.key === ' ') {
      event.preventDefault()
      selection.selectRow(on.id, { isJoining: true, isExtending: false })
    } else if (event.key === 'ContextMenu' || (event.key === 'F10' && event.shiftKey)) {
      event.preventDefault()
      const at = options.getMenuAt(on.id)
      if (at) helpers.requestMenu(on, at)
    }
  }

  function applyStep(step: Step, press: Press): void {
    if (step.turn?.open) tell('open', step.turn.row)
    else if (step.turn) tell('close', step.turn.row)
    if (step.at !== null) selection.selectRow(step.at, press)
    void rows.focusRow(step.at)
  }

  function onKeyDown(event: KeyboardEvent): void {
    takeSelectionKey(event)

    const on = getShownRows().find((row) => row.id === rows.tabbed.value)
    if (!on) return
    takeRowKey(event, on)

    if (!isTreeKey(event.key)) return
    event.preventDefault()

    applyStep(stepTo(getShownRows(), rows.tabbed.value, event.key), {
      isJoining: false,
      isExtending: event.shiftKey,
    })
  }

  return { onKeyDown }
}

export function useTreeGestures(options: TreeGesturesOptions): TreeGesturesState {
  const { getSelected, renamingPath, rows, selection, drag, tell } = options

  function toggleRow(row: ShownRow): void {
    if (row.open) tell('close', row.id)
    else tell('open', row.id)
  }

  function activateRow(row: ShownRow): void {
    if (row.hasChildren) toggleRow(row)
    tell('activate', row.id)
  }

  function requestMenu(row: ShownRow | null, at: Position): void {
    if (row && !selection.picked.value.has(row.id)) selection.selectRow(row.id, PLAIN)
    tell('menu', row?.id ?? null, at)
  }

  function onContextMenu(event: MouseEvent): void {
    requestMenu(null, { x: event.clientX, y: event.clientY })
  }

  function onRowContextMenu(row: ShownRow, event: MouseEvent): void {
    requestMenu(row, { x: event.clientX, y: event.clientY })
  }

  function onRowFocus(row: RowId): void {
    rows.here.value = row
  }

  function onRowPointerDown(row: RowId, event: PointerEvent): void {
    if (event.button !== 0) return
    event.preventDefault()
    ;(event.currentTarget as HTMLElement).focus()

    const how: Press = { isJoining: event.ctrlKey || event.metaKey, isExtending: event.shiftKey }
    selection.hasApplied.value =
      how.isJoining || how.isExtending || !selection.picked.value.has(row)
    const selectedRows = selection.hasApplied.value ? selection.selectRow(row, how) : getSelected()

    drag.lift(getDraggedRows(selectedRows, row), event)
  }

  function onRowClick(row: ShownRow): void {
    const hasApplied = selection.hasApplied.value
    selection.hasApplied.value = false
    if (drag.hasMoved.value || hasApplied) return
    selection.selectRow(row.id, PLAIN)
  }

  function onRowDoubleClick(row: ShownRow): void {
    activateRow(row)
  }

  const { onRename, onAbandon, onFieldBlur } = useTreeRenaming(renamingPath, rows, tell)
  const { onKeyDown } = useTreeKeyboard(options, { activateRow, requestMenu })

  return {
    onContextMenu,
    onRowContextMenu,
    onRowFocus,
    onRowPointerDown,
    onRowClick,
    onRowDoubleClick,
    onRename,
    onAbandon,
    onFieldBlur,
    onKeyDown,
  }
}
