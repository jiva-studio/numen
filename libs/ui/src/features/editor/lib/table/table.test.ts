import { describe, expect, it } from 'vitest'
import { syntaxTree } from '@codemirror/language'
import type { EditorState } from '@codemirror/state'
import { emptyRow, readCell, readTable, getRows, getWidth, withColumn, writeCell } from './table'
import { createState } from '../../fixtures/state'

const getTable = (state: EditorState) => {
  const node = syntaxTree(state).topNode.getChild('Table')
  if (!node) throw new Error('no table was parsed')
  return readTable(state, node)
}

const readItem = (doc: string) => getTable(createState(doc))

const TABLE = `| Word | Kind |
|:-----|----:|
| Entropy | parent |
| Heat | jump |
`

describe('reading a table', () => {
  it('gives every cell the stretch of the text it is written in', () => {
    const state = createState(TABLE)
    const table = getTable(state)
    const cell = table.body[0]?.[0]

    expect(cell).toBeTruthy()
    expect(state.doc.sliceString(cell?.from ?? 0, cell?.to ?? 0)).toBe('Entropy')
  })

  it('reads the head apart from the rows', () => {
    const table = readItem(TABLE)
    expect(table.head.map((cell) => cell?.text)).toEqual(['Word', 'Kind'])
    expect(table.body).toHaveLength(2)
  })

  it('takes the alignment from the row of dashes', () => {
    expect(readItem(TABLE).align).toEqual(['start', 'end'])
    expect(readItem('| a | b |\n|:-:|---|\n| c | d |\n').align).toEqual(['centre', 'none'])
  })

  it('gives an empty cell a place to be written', () => {
    const state = createState('| a | b |\n|---|---|\n| c |  |\n')
    const empty = getTable(state).body[0]?.[1]

    expect(empty).toBeTruthy()
    expect(empty?.from).toBe(empty?.to)
    expect(state.doc.sliceString(0, empty?.from ?? 0)).toBe('| a | b |\n|---|---|\n| c | ')
  })

  it('leaves a column nothing was written in without a cell', () => {
    const table = readItem('| a | b | c |\n|---|---|---|\n| d |\n')
    expect(getWidth(table)).toBe(3)
    expect(getRows(table)[1]).toEqual([expect.objectContaining({ text: 'd' }), null, null])
  })

  it('reads a row that carries no outer pipes', () => {
    const table = readItem('a | b\n---|---\nc | d\n')
    expect(table.head.map((cell) => cell?.text)).toEqual(['a', 'b'])
    expect(table.body[0]?.map((cell) => cell?.text)).toEqual(['c', 'd'])
  })
})

describe('what a cell holds', () => {
  it('escapes a pipe that is typed into it', () => {
    expect(writeCell('a | b')).toBe('a \\| b')
  })

  it('escapes a backslash that is typed into it', () => {
    expect(writeCell('a \\ b')).toBe('a \\\\ b')
  })

  it('folds a line break into a space', () => {
    expect(writeCell('a\nb')).toBe('a b')
  })

  it('shows an escaped pipe as a pipe', () => {
    expect(readCell('a \\| b')).toBe('a | b')
  })

  it('shows an escaped backslash as a backslash', () => {
    expect(readCell('a \\\\ b')).toBe('a \\ b')
  })

  it('preserves backslashes and pipes together', () => {
    const original = 'C:\\path | test'
    expect(readCell(writeCell(original))).toBe(original)
  })

  it('leaves nothing around what was typed', () => {
    expect(writeCell('  a  ')).toBe('a')
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
    const before = readItem(TABLE)
    const after = readItem(`${withColumn(TABLE.trimEnd())}\n`)
    expect(getWidth(after)).toBe(getWidth(before) + 1)
    expect(after.body).toHaveLength(before.body.length)
  })

  it('leaves what was written in the cells alone', () => {
    const after = readItem(`${withColumn(TABLE.trimEnd())}\n`)
    expect(after.head.map((cell) => cell?.text)).toEqual(['Word', 'Kind', ''])
    expect(after.body[0]?.map((cell) => cell?.text)).toEqual(['Entropy', 'parent', ''])
  })
})

describe('a row made at the end', () => {
  it('has a cell for every column', () => {
    expect(emptyRow(3)).toBe('\n| | | |')
  })

  it('parses as a row of that table', () => {
    const table = readItem(`${TABLE.trimEnd()}${emptyRow(2)}\n`)
    expect(table.body).toHaveLength(3)
  })
})

describe('the alignment of a table', () => {
  it('is read from a row of dashes that carries no outer pipes', () => {
    expect(readItem('a | b\n---|:-:\nc | d\n').align).toEqual(['none', 'centre'])
  })

  it('is read from a row of dashes with spaces around its cells', () => {
    expect(readItem('| a | b | c |\n| :-- | --: | :-: |\n| d | e | f |\n').align).toEqual([
      'start',
      'end',
      'centre',
    ])
  })

  it('is read from an indented row of dashes', () => {
    expect(readItem('  | a | b |\n  |:--|--:|\n  | c | d |\n').align).toEqual(['start', 'end'])
  })
})

describe('what a cell reads as', () => {
  it('has no space around it', () => {
    expect(readCell('  a \\| b  ')).toBe('a | b')
  })
})

describe('a column made at the right of rows written unevenly', () => {
  it('keeps a blank line as it is', () => {
    expect(withColumn('| a |\n\n| b |')).toBe('| a | |\n\n| b | |')
  })

  it('closes a row that ends in a pipe once', () => {
    expect(withColumn('a | b |\n---|---|')).toBe('a | b | |\n---|---|---|')
  })

  it('closes a row that begins with a pipe and does not end with one', () => {
    expect(withColumn('| a | b\n|---|---')).toBe('| a | b | |\n|---|---|---|')
  })

  it('ignores the spaces after the last pipe of a row', () => {
    expect(withColumn('| a |   \n|---|\n| b |  ')).toBe('| a | |\n|---|---|\n| b | |')
  })
})
