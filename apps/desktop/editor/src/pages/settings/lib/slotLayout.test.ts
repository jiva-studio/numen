import { describe, expect, it } from 'vitest'
import {
  addSlotSeparator,
  getSeparatorPlace,
  getSlotRows,
  hasSlotLayout,
  moveSlotRow,
  removeSlotSeparator,
  resetSlotRows,
  toggleSlotRow,
  writeSlotRows,
} from './slotLayout'

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
    expect(rows.map((one) => one.kind === 'entry' && one.isHidden)).toStrictEqual([
      false,
      true,
      false,
    ])
  })

  it('move to a place of the list and stay as they were outside it', () => {
    const rows = getSlotRows(group, undefined)

    expect(keys(moveSlotRow(rows, 0, 2))).toStrictEqual([
      'b/view/two',
      'c/view/three',
      'a/view/one',
    ])
    expect(keys(moveSlotRow(rows, 2, 0))).toStrictEqual([
      'c/view/three',
      'a/view/one',
      'b/view/two',
    ])
    expect(moveSlotRow(rows, 0, -1)).toBe(rows)
    expect(moveSlotRow(rows, 2, 3)).toBe(rows)
    expect(moveSlotRow(rows, 1, 1)).toBe(rows)
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

describe('the rows of a slot with groups', () => {
  const grouped = {
    slot: 'titleBar.commands',
    title: 'Actions',
    contributors: [
      { key: 'a/command/one', label: 'one', pluginName: 'A', group: '0-navigate' },
      { key: 'b/command/two', label: 'two', pluginName: 'B', group: '1-new' },
      { key: 'c/command/three', label: 'three', pluginName: 'C', group: '2-settings' },
    ],
  }
  const describeRows = (rows: readonly { kind: string; key: string }[]) =>
    rows.map((one) => (one.kind === 'separator' ? '|' : one.key))

  it('show a separator between the groups when nothing is stored', () => {
    expect(describeRows(getSlotRows(grouped, undefined))).toStrictEqual([
      'a/command/one',
      '|',
      'b/command/two',
      '|',
      'c/command/three',
    ])
  })

  it('follow the stored entries and separators', () => {
    const layout = {
      [grouped.slot]: [{ id: 'c/command/three' }, { id: '---' }, { id: 'a/command/one' }],
    }

    expect(describeRows(getSlotRows(grouped, layout))).toStrictEqual([
      'c/command/three',
      '|',
      'a/command/one',
      'b/command/two',
    ])
  })

  it('pass a row over a separator like a neighbour, and store every separator on the first write, a group left empty included', () => {
    const rows = getSlotRows(grouped, undefined)

    expect(describeRows(moveSlotRow(rows, 0, 1))).toStrictEqual([
      '|',
      'a/command/one',
      'b/command/two',
      '|',
      'c/command/three',
    ])
    expect(writeSlotRows(undefined, grouped.slot, moveSlotRow(rows, 2, 1))).toStrictEqual({
      [grouped.slot]: [
        { id: 'a/command/one' },
        { id: 'b/command/two' },
        { id: '---' },
        { id: '---' },
        { id: 'c/command/three' },
      ],
    })
  })

  it('leave a separator and a hidden state alone when the row at it is toggled', () => {
    const rows = getSlotRows(grouped, undefined)

    expect(toggleSlotRow(rows, 1)).toStrictEqual(rows)
  })
})

describe('the separators a person makes', () => {
  const describeRows = (rows: readonly { kind: string; key: string }[]) =>
    rows.map((one) => (one.kind === 'separator' ? '|' : one.key))

  it('are added after the first row, and numbered apart from those that are there', () => {
    const once = addSlotSeparator(getSlotRows(group, undefined))
    const twice = addSlotSeparator(once)

    expect(describeRows(once)).toStrictEqual(['a/view/one', '|', 'b/view/two', 'c/view/three'])
    expect(describeRows(twice)).toStrictEqual([
      'a/view/one',
      '|',
      '|',
      'b/view/two',
      'c/view/three',
    ])
    expect(new Set(twice.map((one) => one.key)).size).toBe(twice.length)
    expect(addSlotSeparator([])).toStrictEqual([])
  })

  it('are removed alone, and an entry stays', () => {
    const rows = addSlotSeparator(getSlotRows(group, undefined))

    expect(describeRows(removeSlotSeparator(rows, 1))).toStrictEqual([
      'a/view/one',
      'b/view/two',
      'c/view/three',
    ])
    expect(removeSlotSeparator(rows, 0)).toBe(rows)
  })

  it('keep every leading, trailing and repeated one, in the tab and in the stored list', () => {
    const layout = {
      [group.slot]: [
        { id: '---' },
        { id: 'a/view/one' },
        { id: '---' },
        { id: '---' },
        { id: 'b/view/two' },
        { id: '---' },
      ],
    }
    const rows = getSlotRows(group, layout)

    expect(describeRows(rows)).toStrictEqual([
      '|',
      'a/view/one',
      '|',
      '|',
      'b/view/two',
      'c/view/three',
      '|',
    ])
    expect(writeSlotRows(layout, group.slot, rows)).toStrictEqual({
      [group.slot]: [
        { id: '---' },
        { id: 'a/view/one' },
        { id: '---' },
        { id: '---' },
        { id: 'b/view/two' },
        { id: 'c/view/three' },
        { id: '---' },
      ],
    })
  })

  it('keep an entry no list names before the separators that close the list', () => {
    const layout = { [group.slot]: [{ id: 'a/view/one' }, { id: '---' }] }

    expect(describeRows(getSlotRows(group, layout))).toStrictEqual([
      'a/view/one',
      'b/view/two',
      'c/view/three',
      '|',
    ])
  })

  it('move like entries', () => {
    const rows = addSlotSeparator(getSlotRows(group, undefined))

    expect(describeRows(moveSlotRow(rows, 1, 3))).toStrictEqual([
      'a/view/one',
      'b/view/two',
      'c/view/three',
      '|',
    ])
  })
})

describe('the stored list of a slot', () => {
  const layout = { [group.slot]: [{ id: 'a/view/one' }], other: [{ id: 'x' }] }

  it('is there when its key holds a list', () => {
    expect(hasSlotLayout(layout, group.slot)).toBe(true)
    expect(hasSlotLayout(layout, 'missing')).toBe(false)
    expect(hasSlotLayout(undefined, group.slot)).toBe(false)
  })

  it('is reset by deleting its key and leaving the other slots', () => {
    const next = resetSlotRows(layout, group.slot)

    expect(next).toStrictEqual({ other: [{ id: 'x' }] })
    expect('plexGraph.nodeBelow' in (next as object)).toBe(false)
    expect(resetSlotRows(undefined, group.slot)).toStrictEqual({})
  })
})

describe('the separators of a slot moved over one another', () => {
  const layout = {
    [group.slot]: [{ id: '---' }, { id: 'a/view/one' }, { id: '---' }],
  }

  it('are keyed after a move as they are keyed when the stored list is read back', () => {
    const rows = getSlotRows(group, layout)
    const moved = moveSlotRow(rows, 2, 0)

    expect(keys(moved).filter((one) => one.startsWith('separator'))).toStrictEqual([
      'separator:0',
      'separator:1',
    ])
    expect(keys(moved)).toStrictEqual(
      keys(getSlotRows(group, writeSlotRows(layout, group.slot, moved))),
    )
  })

  it('tell each separator its place among the separators', () => {
    const rows = getSlotRows(group, layout)

    expect(getSeparatorPlace(rows, 0)).toStrictEqual({ position: 1, total: 2 })
    expect(getSeparatorPlace(rows, 4)).toStrictEqual({ position: 2, total: 2 })
  })
})
