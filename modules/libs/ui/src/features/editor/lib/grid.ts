/**
 * A table drawn as a table.
 *
 * Where the grid stands is kept on its own element: a widget is told nothing
 * but what it draws.
 */
import type { ChangeDesc, EditorState } from '@codemirror/state'
import { EditorView, WidgetType } from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'
import { listen } from './cells'
import { readCell, readTable, getRows, type Cell, type Row, type Table } from './table'

class Grid extends WidgetType {
  constructor(
    readonly table: Table,
    /** Whether the editor takes typing at all. A read-only table is read. */
    readonly isWritable: boolean,
  ) {
    super()
  }

  override eq(other: Grid) {
    return (
      other.table.from === this.table.from &&
      other.table.source === this.table.source &&
      other.isWritable === this.isWritable
    )
  }

  toDOM(view: EditorView) {
    const frame = document.createElement('div')
    frame.className = 'cm-table'
    this.paint(frame)
    listen(frame, view)
    return frame
  }

  override updateDOM(frame: HTMLElement) {
    const viewElement = [...frame.querySelectorAll<HTMLElement>('.cm-cell')]
    const rows = getRows(this.table)
    const cells = rows.flat()

    if (viewElement.length !== cells.length) {
      this.paint(frame)
      return true
    }

    span(frame, this.table)

    viewElement.forEach((element, index) => {
      const cell = cells[index]
      place(element, cell ?? null, this.isWritable)
      if (cell && element !== document.activeElement) element.textContent = readCell(cell.text)
    })
    return true
  }

  private paint(frame: HTMLElement) {
    const table = document.createElement('table')

    const draw = (row: Row, index: number, tag: 'th' | 'td') => {
      const line = document.createElement('tr')
      line.dataset['row'] = String(index)
      row.forEach((cell, column) => {
        const element = document.createElement(tag)
        element.className = 'cm-cell'
        element.dataset['column'] = String(column)
        const align = this.table.align[column]
        if (align === 'centre') element.style.textAlign = 'center'
        else if (align === 'start' || align === 'end') element.style.textAlign = align
        place(element, cell, this.isWritable)
        if (cell) element.textContent = readCell(cell.text)
        line.appendChild(element)
      })
      return line
    }

    const top = document.createElement('thead')
    const rest = document.createElement('tbody')
    getRows(this.table).forEach((row, index) => {
      if (index === 0) top.appendChild(draw(row, index, 'th'))
      else rest.appendChild(draw(row, index, 'td'))
    })
    table.append(top, rest)

    const expanded = this.isWritable
      ? [createAddButton('cm-add-column'), createAddButton('cm-add-row')]
      : []
    frame.replaceChildren(table, ...expanded)
    span(frame, this.table)
  }
}

/** The one thing a widget cannot read back from the DOM: at its current position. */
const span = (frame: HTMLElement, table: Table) => {
  frame.dataset['from'] = String(table.from)
  frame.dataset['to'] = String(table.to)
}

const createAddButton = (name: string) => {
  const button = document.createElement('button')
  button.className = name
  button.textContent = '+'
  return button
}

/** A cell that is written somewhere can be typed into; one that is not cannot. */
const place = (element: HTMLElement, cell: Cell | null, isWritable: boolean) => {
  if (!cell || !isWritable) {
    element.removeAttribute('contenteditable')
    delete element.dataset['from']
    delete element.dataset['to']
    return
  }
  element.setAttribute('contenteditable', 'plaintext-only')
  element.dataset['from'] = String(cell.from)
  element.dataset['to'] = String(cell.to)
}

/** Whether this editor takes typing at all. */
export const isWritable = (state: EditorState) =>
  !state.readOnly && state.facet(EditorView.editable)

const shiftCells = (row: Row, by: number): Row =>
  row.map((cell) => (cell ? { ...cell, from: cell.from + by, to: cell.to + by } : null))

/** The same widget, standing where the changes have carried the table. Other widgets come back as they are. */
export const moveBlock = (widget: WidgetType, changes: ChangeDesc): WidgetType => {
  if (!(widget instanceof Grid)) return widget
  const table = widget.table
  const by = changes.mapPos(table.from, 1) - table.from
  if (by === 0) return widget
  return new Grid(
    {
      ...table,
      from: table.from + by,
      to: table.to + by,
      head: shiftCells(table.head, by),
      body: table.body.map((row) => shiftCells(row, by)),
    },
    widget.isWritable,
  )
}

/** The widget drawn where a table is written. */
export const getGrid = (state: EditorState, node: SyntaxNode) =>
  new Grid(readTable(state, node), isWritable(state))
