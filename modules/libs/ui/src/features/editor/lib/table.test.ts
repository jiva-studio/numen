import { describe, expect, it } from 'vitest'
import { syntaxTree } from '@codemirror/language'
import type { EditorState } from '@codemirror/state'
import { emptyRow, readTable, rowsOf, shown, widthOf, withColumn, written } from './table'
import { createState } from '../fixtures/state'

const tableIn = (state: EditorState) => {
  const node = syntaxTree(state).topNode.getChild('Table')
  if (!node) throw new Error('no table was parsed')
  return readTable(state, node)
}

const read = (doc: string) => tableIn(createState(doc))

const TABLE = `| Word | Kind |
|:-----|----:|
| Entropy | parent |
| Heat | jump |
`

describe('reading a table', () => {
  it('gives every cell the stretch of the text it is written in', () => {
    const state = createState(TABLE)
    const table = tableIn(state)
    const cell = table.body[0]?.[0]

    expect(cell).toBeTruthy()
    expect(state.doc.sliceString(cell?.from ?? 0, cell?.to ?? 0)).toBe('Entropy')
  })

  it('reads the head apart from the rows', () => {
    const table = read(TABLE)
    expect(table.head.map((cell) => cell?.text)).toEqual(['Word', 'Kind'])
    expect(table.body).toHaveLength(2)
  })

  it('takes the alignment from the row of dashes', () => {
    expect(read(TABLE).align).toEqual(['start', 'end'])
    expect(read('| a | b |\n|:-:|---|\n| c | d |\n').align).toEqual(['centre', 'none'])
  })

  it('gives an empty cell a place to be written', () => {
    const state = createState('| a | b |\n|---|---|\n| c |  |\n')
    const empty = tableIn(state).body[0]?.[1]

    expect(empty).toBeTruthy()
    expect(empty?.from).toBe(empty?.to)
    expect(state.doc.sliceString(0, empty?.from ?? 0)).toBe('| a | b |\n|---|---|\n| c | ')
  })

  it('leaves a column nothing was written in without a cell', () => {
    const table = read('| a | b | c |\n|---|---|---|\n| d |\n')
    expect(widthOf(table)).toBe(3)
    expect(rowsOf(table)[1]).toEqual([expect.objectContaining({ text: 'd' }), null, null])
  })

  it('reads a row that carries no outer pipes', () => {
    const table = read('a | b\n---|---\nc | d\n')
    expect(table.head.map((cell) => cell?.text)).toEqual(['a', 'b'])
    expect(table.body[0]?.map((cell) => cell?.text)).toEqual(['c', 'd'])
  })
})

describe('what a cell holds', () => {
  it('escapes a pipe that is typed into it', () => {
    expect(written('a | b')).toBe('a \\| b')
  })

  it('folds a line break into a space', () => {
    expect(written('a\nb')).toBe('a b')
  })

  it('shows an escaped pipe as a pipe', () => {
    expect(shown('a \\| b')).toBe('a | b')
  })

  it('leaves nothing around what was typed', () => {
    expect(written('  a  ')).toBe('a')
  })
})

describe('a column made at the right', () => {
  it('gives every row an empty cell and the dashes a column', () => {
    expect(withColumn('| a | b |\n|---|---|\n| c | d |')).toBe(
      '| a | b | |\n|---|---|---|\n| c | d | |',
    )
  })

  it('closes a row that carried no pipe at its end', () => {
    expect(withColumn('a | b\n---|---\nc | d')).toBe('a | b | |\n---|---|---|\nc | d | |')
  })

  it('is read back as one column more', () => {
    const before = read(TABLE)
    const after = read(`${withColumn(TABLE.trimEnd())}\n`)
    expect(widthOf(after)).toBe(widthOf(before) + 1)
    expect(after.body).toHaveLength(before.body.length)
  })

  it('leaves what was written in the cells alone', () => {
    const after = read(`${withColumn(TABLE.trimEnd())}\n`)
    expect(after.head.map((cell) => cell?.text)).toEqual(['Word', 'Kind', ''])
    expect(after.body[0]?.map((cell) => cell?.text)).toEqual(['Entropy', 'parent', ''])
  })
})

describe('a row made at the end', () => {
  it('has a cell for every column', () => {
    expect(emptyRow(3)).toBe('\n| | | |')
  })

  it('parses as a row of that table', () => {
    const table = read(`${TABLE.trimEnd()}${emptyRow(2)}\n`)
    expect(table.body).toHaveLength(3)
  })
})
