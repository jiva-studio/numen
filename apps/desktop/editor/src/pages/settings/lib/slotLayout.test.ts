import { describe, expect, it } from 'vitest'
import { getSlotRows, moveSlotRow, toggleSlotRow, writeSlotRows } from './slotLayout'

const group = {
  slot: 'plexGraph.nodeBelow',
  title: 'Below a node',
  contributors: [
    { key: 'a/view/one', label: 'one', pluginName: 'A' },
    { key: 'b/view/two', label: 'two', pluginName: 'B' },
    { key: 'c/view/three', label: 'three', pluginName: 'C' },
  ],
}

const keys = (rows: readonly { key: string }[]) => rows.map((one) => one.key)

describe('the rows of a slot', () => {
  it('follow the order the slot draws them when nothing is stored', () => {
    expect(keys(getSlotRows(group, undefined))).toStrictEqual([
      'a/view/one',
      'b/view/two',
      'c/view/three',
    ])
  })

  it('follow the stored order, with the entries it does not name after them', () => {
    const layout = {
      [group.slot]: [
        { id: 'c/view/three' },
        { id: 'gone/view/x' },
        { id: 'b/view/two', hidden: true },
      ],
    }
    const rows = getSlotRows(group, layout)

    expect(keys(rows)).toStrictEqual(['c/view/three', 'b/view/two', 'a/view/one'])
    expect(rows.map((one) => one.isHidden)).toStrictEqual([false, true, false])
  })

  it('move by one place and stay as they were outside the list', () => {
    const rows = getSlotRows(group, undefined)

    expect(keys(moveSlotRow(rows, 0, 1))).toStrictEqual([
      'b/view/two',
      'a/view/one',
      'c/view/three',
    ])
    expect(moveSlotRow(rows, 0, -1)).toBe(rows)
    expect(moveSlotRow(rows, 2, 3)).toBe(rows)
  })

  it('are stored with the hidden ones marked, and the other slots kept', () => {
    const rows = toggleSlotRow(getSlotRows(group, undefined), 1)

    expect(writeSlotRows({ 'plexGraph.node': [{ id: 'x' }] }, group.slot, rows)).toStrictEqual({
      'plexGraph.node': [{ id: 'x' }],
      [group.slot]: [
        { id: 'a/view/one' },
        { id: 'b/view/two', hidden: true },
        { id: 'c/view/three' },
      ],
    })
  })
})
