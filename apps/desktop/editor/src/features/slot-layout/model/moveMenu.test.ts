/** What the menu that moves an entry lists, and what a choice of it writes. Every plugin here is invented. */
import { describe, expect, it } from 'vitest'
import type { MenuItem } from '@numen/ui'
import type { LayoutEntry, SlotName } from '@numen/plugin-host'
import { getMoveChange, getMoveMenu, type MoveMenuSource } from './moveMenu'

const TIDE = 'titleBar.status|org.example.tide/status'
const REEF = 'titleBar.status|org.example.reef/stats'
const HARBOR = 'titleBar.status|org.example.harbor/update'
const CLOCK = 'statusBar.left|org.example.tick/clock'

const WORDS = { moveTo: 'Move to', atTheEnd: 'At the end' }

const entry = (key: string, title = key): LayoutEntry => ({ key, title })

const createSource = (
  drawn: Partial<Record<SlotName, readonly LayoutEntry[]>>,
  slots: readonly SlotName[] = ['titleBar.status', 'statusBar.left'],
): MoveMenuSource => ({
  listSlots: () => slots,
  listEntries: (slot) => drawn[slot] ?? [],
  getSlotTitle: (slot) => `title of ${slot}`,
})

const describeItems = (items: readonly MenuItem[]): unknown[] =>
  items.map((item) => (item.children ? [item.text, describeItems(item.children)] : item.text))

describe('the menu that moves an entry', () => {
  it('has one item, Move to, with a submenu for each slot', () => {
    const { items } = getMoveMenu(createSource({}), WORDS, TIDE)

    expect(describeItems(items)).toStrictEqual([
      [
        'Move to',
        [
          ['title of titleBar.status', ['At the end']],
          ['title of statusBar.left', ['At the end']],
        ],
      ],
    ])
  })

  it('lists the entries drawn in a slot by their titles, in drawing order, then At the end', () => {
    const { items } = getMoveMenu(
      createSource({
        'titleBar.status': [entry(REEF, 'Index statistics'), entry(HARBOR, 'Update available')],
        'statusBar.left': [entry(CLOCK, 'Clock')],
      }),
      WORDS,
      TIDE,
    )

    expect(describeItems(items)).toStrictEqual([
      [
        'Move to',
        [
          ['title of titleBar.status', ['Index statistics', 'Update available', 'At the end']],
          ['title of statusBar.left', ['Clock', 'At the end']],
        ],
      ],
    ])
  })

  it('does not list the entry that is moved', () => {
    const { items } = getMoveMenu(
      createSource({
        'titleBar.status': [entry(TIDE, 'Tasks'), entry(REEF, 'Index statistics')],
      }),
      WORDS,
      TIDE,
    )

    expect(describeItems(items)).toStrictEqual([
      [
        'Move to',
        [
          ['title of titleBar.status', ['Index statistics', 'At the end']],
          ['title of statusBar.left', ['At the end']],
        ],
      ],
    ])
  })

  it('puts the entry before the entry an item names, and at the end for At the end', () => {
    const { items, targets } = getMoveMenu(
      createSource({ 'statusBar.left': [entry(CLOCK, 'Clock')] }),
      WORDS,
      TIDE,
    )
    const left = items[0]?.children?.[1]
    const [clock, end] = left?.children ?? []

    expect(targets.get(clock?.id ?? '')).toStrictEqual({ slot: 'statusBar.left', before: CLOCK })
    expect(targets.get(end?.id ?? '')).toStrictEqual({ slot: 'statusBar.left' })
    expect(targets.get(left?.id ?? '')).toBeUndefined()
    expect(targets.get(items[0]?.id ?? '')).toBeUndefined()
  })

  it('gives every item an id of its own', () => {
    const { items } = getMoveMenu(
      createSource({ 'titleBar.status': [entry(REEF), entry(HARBOR)] }),
      WORDS,
      TIDE,
    )
    const ids = JSON.stringify(items).match(/"id":"[^"]*"/g) ?? []

    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('the write of a choice of the menu', () => {
  const drawn: Partial<Record<SlotName, readonly string[]>> = {
    'titleBar.status': [TIDE, REEF, HARBOR],
    'statusBar.left': [CLOCK],
  }
  const listKeys = (slot: SlotName): readonly string[] => drawn[slot] ?? []

  it('holds both lists of a move into another slot', () => {
    expect(
      getMoveChange({}, listKeys, {
        key: HARBOR,
        from: 'titleBar.status',
        target: { slot: 'statusBar.left', before: CLOCK },
      }),
    ).toStrictEqual({
      'titleBar.status': [TIDE, REEF],
      'statusBar.left': [HARBOR, CLOCK],
    })
  })

  it('holds the one list of a move within a slot', () => {
    expect(
      getMoveChange({}, listKeys, {
        key: HARBOR,
        from: 'titleBar.status',
        target: { slot: 'titleBar.status', before: TIDE },
      }),
    ).toStrictEqual({ 'titleBar.status': [HARBOR, TIDE, REEF] })
  })

  it('puts the entry last for a target with no key', () => {
    expect(
      getMoveChange({}, listKeys, {
        key: TIDE,
        from: 'titleBar.status',
        target: { slot: 'titleBar.status' },
      }),
    ).toStrictEqual({ 'titleBar.status': [REEF, HARBOR, TIDE] })
  })

  it('is empty for a move that changes nothing', () => {
    const none = { key: REEF, from: 'titleBar.status' as const }

    expect(
      getMoveChange({}, listKeys, { ...none, target: { slot: 'titleBar.status', before: HARBOR } }),
    ).toStrictEqual({})
    expect(
      getMoveChange({}, listKeys, {
        key: HARBOR,
        from: 'titleBar.status',
        target: { slot: 'titleBar.status' },
      }),
    ).toStrictEqual({})
  })
})
