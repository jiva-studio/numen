/** Typing into a drawn table: what a cell writes back, and what the keys do. */
import { EditorView } from '@codemirror/view'
import { emptyRow, withColumn, writeCell } from './table'

const getSpan = (frame: HTMLElement) => {
  const from = Number(frame.dataset['from'])
  const to = Number(frame.dataset['to'])
  return Number.isNaN(from) || Number.isNaN(to) ? null : { from, to }
}

const getCells = (frame: HTMLElement) => [
  ...frame.querySelectorAll<HTMLElement>('.cm-cell[data-from]'),
]

const getRange = (element: HTMLElement) => {
  const from = Number(element.dataset['from'])
  const to = Number(element.dataset['to'])
  return Number.isNaN(from) || Number.isNaN(to) ? null : { from, to }
}

/** What is in the cell, written back over the stretch the cell came from. */
const keep = (view: EditorView, element: HTMLElement) => {
  const range = getRange(element)
  if (!range) return
  const text = writeCell(element.textContent ?? '')
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

const countColumns = (frame: HTMLElement) => frame.querySelectorAll('thead .cm-cell').length

/** A row under the table, and a column at its right. */
const grow = (frame: HTMLElement, view: EditorView) => {
  const where = getSpan(frame)
  if (!where) return null
  return {
    row: () => ({
      changes: {
        from: view.state.doc.lineAt(where.to).to,
        insert: emptyRow(countColumns(frame)),
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
export const listen = (frame: HTMLElement, view: EditorView) => {
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
    const text = writeCell(clipboard.getData('text/plain'))
    frame.ownerDocument.execCommand('insertText', false, text)
  })

  frame.addEventListener('keydown', (event) => {
    handleCellKeydown(event, frame, view)
  })
}

function handleCellKeydown(event: KeyboardEvent, frame: HTMLElement, view: EditorView): void {
  const cell = (event.target as HTMLElement).closest<HTMLElement>('.cm-cell')
  if (!cell) return

  if (event.key === 'Escape') {
    event.preventDefault()
    const range = getRange(cell)
    if (range) view.dispatch({ selection: { anchor: view.state.doc.lineAt(range.to).to } })
    view.focus()
    return
  }

  if (event.key === 'Enter') {
    event.preventDefault()
    const cells = getCells(frame)
    const columns = countColumns(frame)
    const below = cells[cells.indexOf(cell) + columns]
    if (below) focus(below)
    return
  }

  if (event.key !== 'Tab') return
  event.preventDefault()
  const cells = getCells(frame)
  const pos = cells.indexOf(cell)

  if (event.shiftKey) {
    focus(cells[Math.max(pos - 1, 0)])
    return
  }
  if (pos < cells.length - 1) {
    focus(cells[pos + 1])
    return
  }

  const range = getRange(cell)
  if (!range) return
  const end = view.state.doc.lineAt(range.to).to
  const columns = countColumns(frame)
  view.dispatch({ changes: { from: end, insert: emptyRow(columns) } })
  focus(getCells(frame)[pos + 1])
}
