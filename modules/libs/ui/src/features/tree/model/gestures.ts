/**
 * What a press, a double press, a key or a menu asked for on a row comes to.
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

export function useTreeGestures(options: TreeGesturesOptions): TreeGesturesState {
  const { getShownRows, getSelected, renamingPath, rows, selection, drag, tell } = options

  function toggleRow(row: ShownRow): void {
    if (row.open) tell('close', row.id)
    else tell('open', row.id)
  }

  function activateRow(row: ShownRow): void {
    if (row.hasChildren) toggleRow(row)
    tell('activate', row.id)
  }

  /** A menu asked for on a row, which the selection takes in first, or off every row. */
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

    const how: Press = { joining: event.ctrlKey || event.metaKey, reaching: event.shiftKey }
    selection.said.value = how.joining || how.reaching || !selection.picked.value.has(row)
    const taken = selection.said.value ? selection.selectRow(row, how) : getSelected()

    drag.lift(getDraggedRows(taken, row), event)
  }

  function onRowClick(row: ShownRow): void {
    const spoken = selection.said.value
    selection.said.value = false
    if (drag.hasMoved.value || spoken) return
    selection.selectRow(row.id, PLAIN)
  }

  function onRowDoubleClick(row: ShownRow): void {
    activateRow(row)
  }

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

  /** The keys that act on the selection, told apart from the rows. Tells whether the key was taken. */
  function takeSelectionKey(event: KeyboardEvent): boolean {
    const chorded = event.ctrlKey || event.metaKey

    if (chorded && event.key.toLowerCase() === 'a') {
      event.preventDefault()
      selection.selectEveryRow()
      return true
    }

    if (event.key === 'Delete' || event.key === 'Backspace') {
      event.preventDefault()
      if (getSelected().length > 0) tell('remove', getSelected())
      return true
    }

    return false
  }

  /** The keys that act on the row the keyboard stands on. Tells whether the key was taken. */
  function takeRowKey(event: KeyboardEvent, on: ShownRow): boolean {
    if (event.key === 'Enter') {
      event.preventDefault()
      activateRow(on)
      return true
    }

    // The row the keyboard stands on joins the selection, or leaves it.
    if (event.key === ' ') {
      event.preventDefault()
      selection.selectRow(on.id, { joining: true, reaching: false })
      return true
    }

    if (event.key === 'ContextMenu' || (event.key === 'F10' && event.shiftKey)) {
      event.preventDefault()
      const at = options.getMenuAt(on.id)
      if (at) requestMenu(on, at)
      return true
    }

    return false
  }

  /** The turn a step asks for, and the row it lands on. */
  function applyStep(step: Step, press: Press): void {
    if (step.turn?.open) tell('open', step.turn.row)
    else if (step.turn) tell('close', step.turn.row)
    if (step.at !== null) selection.selectRow(step.at, press)
    void rows.focusRow(step.at)
  }

  function onKeyDown(event: KeyboardEvent): void {
    if (takeSelectionKey(event)) return

    const on = getShownRows().find((row) => row.id === rows.tabbed.value)
    if (!on) return
    if (takeRowKey(event, on)) return

    if (!isTreeKey(event.key)) return
    event.preventDefault()

    applyStep(stepTo(getShownRows(), rows.tabbed.value, event.key), {
      joining: false,
      reaching: event.shiftKey,
    })
  }

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
