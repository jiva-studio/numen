import { describe, expect, it } from 'vitest'
import { syntaxTree } from '@codemirror/language'
import { ChangeSet, EditorState, type Extension } from '@codemirror/state'
import { EditorView, WidgetType } from '@codemirror/view'
import { gridOf, isWritable, moveBlock } from './grid'
import { createState } from '../fixtures/state'

const TABLE = '| a | b |\n|---|---|\n| c | d |\n'

const createGrid = (doc: string) => {
  const state = createState(doc)
  const node = syntaxTree(state).topNode.getChild('Table')
  if (!node) throw new Error('no table was parsed')
  return gridOf(state, node)
}

describe('whether the editor takes typing', () => {
  const createStateWith = (...extensions: Extension[]) => EditorState.create({ extensions })

  it('does where it is neither read-only nor closed', () => {
    expect(isWritable(createStateWith())).toBe(true)
  })

  it('does not where it is read-only', () => {
    expect(isWritable(createStateWith(EditorState.readOnly.of(true)))).toBe(false)
  })

  it('does not where it is not editable', () => {
    expect(isWritable(createStateWith(EditorView.editable.of(false)))).toBe(false)
  })

  it('does not where it is both', () => {
    const state = createStateWith(EditorState.readOnly.of(true), EditorView.editable.of(false))
    expect(isWritable(state)).toBe(false)
  })
})

describe('a grid carried by changes', () => {
  it('is the same widget where nothing moved the table', () => {
    const grid = createGrid(TABLE)
    const changes = ChangeSet.of({ from: TABLE.length, insert: 'after' }, TABLE.length)
    expect(moveBlock(grid, changes)).toBe(grid)
  })

  it('stands where the changes carried it', () => {
    const grid = createGrid(TABLE)
    const changes = ChangeSet.of({ from: 0, insert: 'xx' }, TABLE.length)
    const moved = moveBlock(grid, changes) as unknown as typeof grid

    expect(moved).not.toBe(grid)
    expect(moved.table.from).toBe(grid.table.from + 2)
    expect(moved.table.to).toBe(grid.table.to + 2)
    expect(moved.table.head.map((cell) => cell?.from)).toEqual(
      grid.table.head.map((cell) => (cell?.from ?? 0) + 2),
    )
    expect(moved.table.body[0]?.map((cell) => cell?.to)).toEqual(
      grid.table.body[0]?.map((cell) => (cell?.to ?? 0) + 2),
    )
    expect(moved.isWritable).toBe(grid.isWritable)
  })

  it('is any other widget as it was', () => {
    const other = new (class extends WidgetType {
      toDOM() {
        return document.createElement('span')
      }
    })()
    expect(moveBlock(other, ChangeSet.of({ from: 0, insert: 'x' }, 3))).toBe(other)
  })
})
