/**
 * A table, drawn as a table and typed into cell by cell.
 *
 * Reading gives every cell the stretch of the file it is written in; a cell
 * that is typed into writes back over that stretch alone, so the rest of the
 * table stays byte for byte as the person left it. A cell holds one line: a
 * pipe typed into one is escaped, a line break becomes a space.
 */
import type { EditorState } from '@codemirror/state'
import { EditorView, WidgetType } from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'

export type Align = 'none' | 'start' | 'centre' | 'end'

/** One cell, and where it is written. */
export interface Cell {
  readonly from: number
  readonly to: number
  readonly text: string
}

/** A row can be short. Where a column has nothing written, it has no cell. */
export type Row = readonly (Cell | null)[]

export interface Table {
  readonly from: number
  readonly to: number
  readonly source: string
  readonly align: readonly Align[]
  readonly head: Row
  readonly body: readonly Row[]
}

const ALIGNMENT = /^:?-+:?$/

const alignmentOf = (text: string): Align[] => {
  const parts = text.split('|')
  if (parts[0]?.trim() === '') parts.shift()
  if (parts[parts.length - 1]?.trim() === '') parts.pop()
  return parts.map((part) => {
    const written = part.trim()
    if (!ALIGNMENT.test(written)) return 'none'
    const start = written.startsWith(':')
    const end = written.endsWith(':')
    if (start && end) return 'centre'
    if (start) return 'start'
    if (end) return 'end'
    return 'none'
  })
}

const cellsOf = (state: EditorState, row: SyntaxNode): Cell[] => {
  const cells: Cell[] = []
  let after: number | null = null
  let written = false

  for (let child = row.firstChild; child; child = child.nextSibling) {
    if (child.name === 'TableCell') {
      cells.push({ from: child.from, to: child.to, text: state.doc.sliceString(child.from, child.to) })
      written = true
      continue
    }
    if (child.name !== 'TableDelimiter') continue
    if (after !== null && !written) {
      const at = Math.min(after + 1, child.from)
      cells.push({ from: at, to: at, text: '' })
    }
    after = child.to
    written = false
  }
  return cells
}

/** What the table under `node` says, and where every part of it is written. */
export const readTable = (state: EditorState, node: SyntaxNode): Table => {
  let head: Cell[] = []
  const body: Cell[][] = []
  let align: Align[] = []

  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.name === 'TableHeader') head = cellsOf(state, child)
    else if (child.name === 'TableRow') body.push(cellsOf(state, child))
    else if (child.name === 'TableDelimiter')
      align = alignmentOf(state.doc.sliceString(child.from, child.to))
  }

  return {
    from: node.from,
    to: node.to,
    source: state.doc.sliceString(node.from, node.to),
    align,
    head,
    body,
  }
}

/** How many columns the table has, counting the widest row. */
export const widthOf = (table: Table): number =>
  Math.max(table.align.length, table.head.length, ...table.body.map((row) => row.length), 1)

/** Every row, head first, each as wide as the table. */
export const rowsOf = (table: Table): Row[] => {
  const width = widthOf(table)
  const pad = (row: Row): Row =>
    Array.from({ length: width }, (_, column) => row[column] ?? null)
  return [pad(table.head), ...table.body.map(pad)]
}

/** What a cell reads as. */
export const shown = (text: string): string => text.replace(/\\\|/g, '|').trim()

/** What typing in a cell writes. */
export const written = (text: string): string =>
  text.replace(/\s*\n\s*/g, ' ').replace(/\|/g, '\\|').trim()

/** A row of empty cells, written under the table. */
export const emptyRow = (width: number): string => `\n|${' |'.repeat(width)}`

/**
 * The table with one more column.
 *
 * Every line gains an empty cell at its end, and the row of dashes gains a
 * column of dashes. A line that carried no pipe at its end gets one, which is
 * the only way an empty last cell can be written.
 */
export const withColumn = (source: string): string =>
  source
    .split('\n')
    .map((line, index) => {
      const dashes = index === 1
      const written = line.trimEnd()
      if (!written) return line
      const closed = written.endsWith('|') ? written : `${written}${dashes ? '|' : ' |'}`
      return dashes ? `${closed}---|` : `${closed} |`
    })
    .join('\n')

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
