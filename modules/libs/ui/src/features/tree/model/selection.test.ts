/** What a press on a row leaves selected, and when the owner is told. */
import { describe, expect, it, vi } from 'vitest'
import { useRowSelection } from './selection'
import { PLAIN } from '../lib/select'
import type { RowId, ShownRow } from '../lib/row'

const createRow = (id: RowId): ShownRow => ({
  id,
  name: id,
  hasChildren: false,
  parent: null,
  level: 1,
  isLast: false,
  hasRows: false,
  open: false,
})

const ROWS = ['a', 'b', 'c', 'd'].map(createRow)

/** The selection the owner holds, changed only by what it is told. */
function createOwner(initial: readonly RowId[] = []) {
  let target = initial
  const select = vi.fn<(rows: readonly RowId[]) => void>((rows) => {
    target = rows
  })
  const state = useRowSelection(
    () => ROWS,
    () => target,
    select,
  )
  return { state, select }
}

describe('a press on a row', () => {
  it('has applied nothing before any press', () => {
    expect(createOwner().state.hasApplied.value).toBe(false)
  })

  it('selects the row and notifies the owner', () => {
    const { state, select } = createOwner()

    expect(state.selectRow('b', PLAIN)).toStrictEqual(['b'])
    expect(select).toHaveBeenCalledExactlyOnceWith(['b'])
    expect(state.picked.value.has('b')).toBe(true)
  })

  it('notifies the owner nothing when the selection is the same', () => {
    const { state, select } = createOwner(['b'])

    state.selectRow('b', PLAIN)

    expect(select).not.toHaveBeenCalled()
  })

  it('reaches from the row a plain press last landed on', () => {
    const { state } = createOwner()

    state.selectRow('b', PLAIN)

    expect(state.selectRow('d', { isJoining: false, isExtending: true })).toStrictEqual([
      'b',
      'c',
      'd',
    ])
  })

  it('selects every row drawn', () => {
    const { state, select } = createOwner()

    expect(state.selectEveryRow()).toStrictEqual(['a', 'b', 'c', 'd'])
    expect(select).toHaveBeenCalledExactlyOnceWith(['a', 'b', 'c', 'd'])
  })

  it('reaches from the first row after every row was selected', () => {
    const { state } = createOwner()

    state.selectEveryRow()

    expect(state.selectRow('c', { isJoining: false, isExtending: true })).toStrictEqual([
      'a',
      'b',
      'c',
    ])
  })
})
