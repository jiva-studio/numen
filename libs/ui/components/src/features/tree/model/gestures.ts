import type { Ref } from 'vue'
import { getDraggedRows } from '../lib/drag'
import type { RowId, ShownRow } from '../lib/row'
import { PLAIN, type Press } from '../lib/select'
import { isTreeKey, stepTo, type Step } from '../lib/step'
import { runInBackground } from '@/shared/lib/async/background'
import { reportFailure } from '@/shared/api/report'
import type { Position } from '@/shared/lib/math/geometry'
import type { RowDragState } from './drag'
import type { DrawnRowsState } from './rows'
import type { RowSelectionState } from './selection'

/** Emits tree gesture events. */
export interface TreeGesturesEmit {
  (event: 'open', row: RowId): void
  (event: 'close', row: RowId): void
  (event: 'activate', row: RowId): void
  (event: 'rename', row: RowId, name: string): void
  (event: 'remove', rows: readonly RowId[]): void
  (event: 'menu', row: RowId | null, pos: Position): void
}

/** Options for tree pointer and keyboard gesture handlers. */
export interface TreeGesturesOptions {
  readonly getShownRows: () => readonly ShownRow[]
  readonly getSelected: () => readonly RowId[]
  readonly rows: DrawnRowsState
  readonly selection: RowSelectionState
  readonly drag: RowDragState
  readonly getMenuAt: (row: RowId) => Position | null
  readonly emit: TreeGesturesEmit
}

/** Event handlers for tree pointer and keyboard gestures. */
export interface TreeGesturesState {
  readonly onContextMenu: (event: MouseEvent) => void
  readonly onRowContextMenu: (row: ShownRow, event: MouseEvent) => void
  readonly onRowFocus: (row: RowId) => void
  readonly onRowPointerDown: (row: RowId, event: PointerEvent) => void
  readonly onRowClick: (row: ShownRow) => void
  readonly onRowDoubleClick: (row: ShownRow) => void
  readonly onKeyDown: (event: KeyboardEvent) => void
}

function focusInBackground(rows: DrawnRowsState, row: RowId | null): void {
  runInBackground(rows.focusRow(row), reportFailure)
}

/** Creates the handlers that end the rename of a row. */
export function createTreeRenaming(
  renamingPath: Ref<RowId | null>,
  rows: DrawnRowsState,
  emit: TreeGesturesEmit,
) {
  function onRename(row: RowId, name: string): void {
    renamingPath.value = null
    emit('rename', row, name)
    focusInBackground(rows, row)
  }

  function onAbandon(row: RowId): void {
    renamingPath.value = null
    focusInBackground(rows, row)
  }

  function onFieldBlur(): void {
    renamingPath.value = null
  }

  return { onRename, onAbandon, onFieldBlur }
}

interface TreeKeyboardHelpers {
  readonly activateRow: (row: ShownRow) => void
  readonly requestMenu: (row: ShownRow | null, pos: Position) => void
}

function createTreeKeyboard(options: TreeGesturesOptions, helpers: TreeKeyboardHelpers) {
  const { getShownRows, getSelected, rows, selection, emit } = options

  function takeSelectionKey(event: KeyboardEvent): void {
    const chorded = event.ctrlKey || event.metaKey
    if (chorded && event.key.toLowerCase() === 'a') {
      event.preventDefault()
      selection.selectEveryRow()
    } else if (event.key === 'Delete' || event.key === 'Backspace') {
      event.preventDefault()
      if (getSelected().length > 0) emit('remove', getSelected())
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
      const menuPos = options.getMenuAt(on.id)
      if (menuPos) helpers.requestMenu(on, menuPos)
    }
  }

  function applyStep(step: Step, press: Press): void {
    if (step.turn?.isOpen) emit('open', step.turn.row)
    else if (step.turn) emit('close', step.turn.row)
    // Stryker disable next-line ConditionalExpression: step.at is non-null whenever getShownRows() has a tabbed row
    if (step.at !== null) selection.selectRow(step.at, press)
    focusInBackground(rows, step.at)
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

/** Creates pointer, keyboard, and rename event handlers for a tree. */
export function createTreeGestures(options: TreeGesturesOptions): TreeGesturesState {
  const { getSelected, rows, selection, drag, emit } = options

  function toggleRow(row: ShownRow): void {
    if (row.isOpen) emit('close', row.id)
    else emit('open', row.id)
  }

  function activateRow(row: ShownRow): void {
    if (row.hasChildren) toggleRow(row)
    emit('activate', row.id)
  }

  function requestMenu(row: ShownRow | null, pos: Position): void {
    if (row && !selection.picked.value.has(row.id)) selection.selectRow(row.id, PLAIN)
    emit('menu', row?.id ?? null, pos)
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

    drag.startPress(getDraggedRows(selectedRows, row), event)
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

  const { onKeyDown } = createTreeKeyboard(options, { activateRow, requestMenu })

  return {
    onContextMenu,
    onRowContextMenu,
    onRowFocus,
    onRowPointerDown,
    onRowClick,
    onRowDoubleClick,
    onKeyDown,
  }
}
