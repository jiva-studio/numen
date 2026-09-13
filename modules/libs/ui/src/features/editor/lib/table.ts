/**
 * A table, as it is written.
 *
 * Reading gives every cell the stretch of the file it is written in, so what
 * is typed into a cell can be written back over that stretch alone. A cell
 * holds one line: a pipe in one is escaped, a line break becomes a space.
 */
import type { EditorState } from '@codemirror/state'
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
      cells.push({
        from: child.from,
        to: child.to,
        text: state.doc.sliceString(child.from, child.to),
      })
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
  const pad = (row: Row): Row => Array.from({ length: width }, (_, column) => row[column] ?? null)
  return [pad(table.head), ...table.body.map(pad)]
}

/** What a cell reads as. */
export const readCell = (text: string): string => text.replace(/\\\|/g, '|').trim()

/** What typing in a cell writes. */
export const writeCell = (text: string): string =>
  text
    .replace(/\s*\n\s*/g, ' ')
    .replace(/\|/g, '\\|')
    .trim()

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
