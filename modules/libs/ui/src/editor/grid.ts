/**
 * A table drawn as a table and typed into cell by cell.
 *
 * A cell writes back over the stretch it was read from, so the rest of the
 * table stays byte for byte as the person left it. Where the grid stands is
 * kept on its own element: a widget is told nothing but what it draws.
 */
import type { EditorState } from '@codemirror/state'
import { EditorView, WidgetType } from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'
import {
  emptyRow,
  readTable,
  rowsOf,
  shown,
  withColumn,
  written,
  type Cell,
  type Row,
  type Table,
} from './table'

class Grid extends WidgetType {
  constructor(
    readonly table: Table,
    /** Whether the editor takes typing at all. A read-only table is read. */
    readonly writable: boolean,
  ) {
    super()
  }

  override eq(other: Grid) {
    return (
      other.table.from === this.table.from &&
      other.table.source === this.table.source &&
      other.writable === this.writable
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
    const drawn = [...frame.querySelectorAll<HTMLElement>('.cm-cell')]
    const rows = rowsOf(this.table)
    const cells = rows.flat()

    if (drawn.length !== cells.length) {
      this.paint(frame)
      return true
    }

    span(frame, this.table)

    drawn.forEach((element, index) => {
      const cell = cells[index]
      place(element, cell ?? null, this.writable)
      if (cell && element !== document.activeElement) element.textContent = shown(cell.text)
    })
    return true
  }

  private paint(frame: HTMLElement) {
    const [head, ...body] = rowsOf(this.table)
    const table = document.createElement('table')

    const draw = (row: Row, index: number, tag: 'th' | 'td') => {
      const line = document.createElement('tr')
      line.dataset['row'] = String(index)
      row.forEach((cell, column) => {
        const element = document.createElement(tag)
        element.className = 'cm-cell'
        element.dataset['column'] = String(column)
        const align = this.table.align[column] ?? 'none'
        if (align !== 'none') element.style.textAlign = align === 'centre' ? 'center' : align
        place(element, cell, this.writable)
        if (cell) element.textContent = shown(cell.text)
        line.appendChild(element)
      })
      return line
    }

    const top = document.createElement('thead')
    top.appendChild(draw(head ?? [], 0, 'th'))
    const rest = document.createElement('tbody')
    body.forEach((row, index) => rest.appendChild(draw(row, index + 1, 'td')))
    table.append(top, rest)

    const grown = this.writable ? [adding('cm-add-column'), adding('cm-add-row')] : []
    frame.replaceChildren(table, ...grown)
    span(frame, this.table)
  }
}

/** The one thing a widget cannot read back from the DOM: where it stands. */
const span = (frame: HTMLElement, table: Table) => {
  frame.dataset['from'] = String(table.from)
  frame.dataset['to'] = String(table.to)
}

const spanOf = (frame: HTMLElement) => {
  const from = Number(frame.dataset['from'])
  const to = Number(frame.dataset['to'])
  return Number.isNaN(from) || Number.isNaN(to) ? null : { from, to }
}

const adding = (name: string) => {
  const button = document.createElement('button')
  button.className = name
  button.textContent = '+'
  return button
}

/** A cell that is written somewhere can be typed into; one that is not cannot. */
const place = (element: HTMLElement, cell: Cell | null, writable: boolean) => {
  if (!cell || !writable) {
    element.removeAttribute('contenteditable')
    delete element.dataset['from']
    delete element.dataset['to']
    return
  }
  element.setAttribute('contenteditable', 'plaintext-only')
  element.dataset['from'] = String(cell.from)
  element.dataset['to'] = String(cell.to)
}

const cellsIn = (frame: HTMLElement) =>
  [...frame.querySelectorAll<HTMLElement>('.cm-cell[data-from]')]

const rangeOf = (element: HTMLElement) => {
  const from = Number(element.dataset['from'])
  const to = Number(element.dataset['to'])
  return Number.isNaN(from) || Number.isNaN(to) ? null : { from, to }
}

/** What is in the cell, written back over the stretch the cell came from. */
const keep = (view: EditorView, element: HTMLElement) => {
  const range = rangeOf(element)
  if (!range) return
  const text = written(element.textContent ?? '')
  if (view.state.doc.sliceString(range.from, range.to) === text) return
  view.dispatch({ changes: { from: range.from, to: range.to, insert: text } })
}

const focus = (element: HTMLElement | undefined) => {
  if (!element) return
  element.focus()
  const selection = element.ownerDocument.defaultView?.getSelection()
  if (!selection) return
  const range = element.ownerDocument.createRange()
  range.selectNodeContents(element)
  range.collapse(false)
  selection.removeAllRanges()
  selection.addRange(range)
}

const columnsIn = (frame: HTMLElement) => frame.querySelectorAll('thead .cm-cell').length

/** A row under the table, and a column at its right. */
const grow = (frame: HTMLElement, view: EditorView) => {
  const where = spanOf(frame)
  if (!where) return null
  return {
    row: () => ({
      changes: {
        from: view.state.doc.lineAt(where.to).to,
        insert: emptyRow(columnsIn(frame)),
      },
    }),
    column: () => ({
      changes: {
        from: where.from,
        to: where.to,
        insert: withColumn(view.state.doc.sliceString(where.from, where.to)),
      },
    }),
  }
}

/** Tab walks the cells; the last one makes a row. Escape leaves the table. */
const listen = (frame: HTMLElement, view: EditorView) => {
  frame.addEventListener('input', (event) => {
    const cell = (event.target as HTMLElement).closest<HTMLElement>('.cm-cell')
    if (cell) keep(view, cell)
  })

  frame.addEventListener('mousedown', (event) => {
    if ((event.target as HTMLElement).closest('button')) event.preventDefault()
  })

  frame.addEventListener('click', (event) => {
    const button = (event.target as HTMLElement).closest('button')
    if (!button) return
    const more = grow(frame, view)
    if (!more) return
    if (button.classList.contains('cm-add-row')) view.dispatch(more.row())
    if (button.classList.contains('cm-add-column')) view.dispatch(more.column())
  })

  frame.addEventListener('paste', (event) => {
    const clipboard = (event as ClipboardEvent).clipboardData
    if (!clipboard) return
    event.preventDefault()
    const text = written(clipboard.getData('text/plain'))
    frame.ownerDocument.execCommand('insertText', false, text)
  })

  frame.addEventListener('keydown', (event) => {
    const cell = (event.target as HTMLElement).closest<HTMLElement>('.cm-cell')
    if (!cell) return

    if (event.key === 'Escape') {
      event.preventDefault()
      const range = rangeOf(cell)
      if (range) view.dispatch({ selection: { anchor: view.state.doc.lineAt(range.to).to } })
      view.focus()
      return
    }

    if (event.key === 'Enter') {
      event.preventDefault()
      const cells = cellsIn(frame)
      const columns = columnsIn(frame)
      const below = cells[cells.indexOf(cell) + columns]
      if (below) focus(below)
      return
    }

    if (event.key !== 'Tab') return
    event.preventDefault()
    const cells = cellsIn(frame)
    const at = cells.indexOf(cell)

    if (event.shiftKey) {
      focus(cells[Math.max(at - 1, 0)])
      return
    }
    if (at < cells.length - 1) {
      focus(cells[at + 1])
      return
    }

    const range = rangeOf(cell)
    if (!range) return
    const end = view.state.doc.lineAt(range.to).to
    const columns = columnsIn(frame)
    view.dispatch({ changes: { from: end, insert: emptyRow(columns) } })
    focus(cellsIn(frame)[at + 1])
  })
}

/** Whether this editor takes typing at all. */
export const writable = (state: EditorState) => !state.readOnly && state.facet(EditorView.editable)

/** The widget drawn where a table is written. */
export const gridOf = (state: EditorState, node: SyntaxNode) =>
  new Grid(readTable(state, node), writable(state))
