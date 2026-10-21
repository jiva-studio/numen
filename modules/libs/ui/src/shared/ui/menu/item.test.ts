/**
 * Where a menu goes, and where the keyboard goes in it. Plain values, so the
 * awkward cases are cheap: a menu at the far corner, and one bigger than the
 * area it has to fit in.
 */
import { describe, expect, it } from 'vitest'
import { filterItems, groupItems, getInitialIndex, placeMenu, type MenuItem } from './item'

const VIEWPORT = { width: 1000, height: 800 }
const SIZE = { width: 200, height: 300 }

const getMenuPosition = (
  x: number,
  y: number,
  over: Partial<Parameters<typeof placeMenu>[0]> = {},
) => placeMenu({ at: { x, y }, size: SIZE, viewport: VIEWPORT, margin: 8, ...over })

describe('where a menu goes', () => {
  it('runs on from the point it was requested for', () => {
    expect(getMenuPosition(100, 120)).toStrictEqual({ x: 100, y: 120 })
  })

  it('runs back over the point when the far edge is nearer than its length', () => {
    expect(getMenuPosition(900, 700)).toStrictEqual({ x: 700, y: 400 })
  })

  it('folds one way and not the other, each edge on its own', () => {
    expect(getMenuPosition(900, 100)).toStrictEqual({ x: 700, y: 100 })
    expect(getMenuPosition(100, 700)).toStrictEqual({ x: 100, y: 400 })
  })

  it('is brought in off an edge it was requested for right against', () => {
    expect(getMenuPosition(2, 1)).toStrictEqual({ x: 8, y: 8 })
  })

  it('is brought in where folding back would take it off the near edge', () => {
    // Nothing fits either side of the point, so it is shifted rather than
    // folded, and the whole of it is on screen.
    const placed = placeMenu({
      at: { x: 120, y: 120 },
      size: SIZE,
      viewport: { width: 260, height: 340 },
      margin: 8,
    })
    expect(placed).toStrictEqual({ x: 52, y: 32 })
  })

  it('sits at the near edge when it is larger than the area it is placed in', () => {
    const placed = placeMenu({
      at: { x: 40, y: 40 },
      size: SIZE,
      viewport: { width: 160, height: 200 },
      margin: 8,
    })
    expect(placed).toStrictEqual({ x: 8, y: 8 })
  })

  it('keeps whatever clearance it was given', () => {
    expect(getMenuPosition(0, 0, { margin: 24 })).toStrictEqual({ x: 24, y: 24 })
    expect(getMenuPosition(0, 0, { margin: 0 })).toStrictEqual({ x: 0, y: 0 })
  })
})

describe('where the keyboard lands as a menu opens', () => {
  const items: MenuItem[] = [
    { id: 'one', text: 'One', disabled: true },
    { id: 'two', text: 'Two' },
  ]

  it('is the first item that can be chosen, for one opened from the keyboard', () => {
    expect(getInitialIndex('keyboard', items)).toBe(1)
  })

  it('is no item at all, for one opened by hand', () => {
    expect(getInitialIndex('pointer', items)).toBe(-1)
  })

  it('is no item at all either way when there is nothing to land on', () => {
    expect(getInitialIndex('keyboard', [])).toBe(-1)
    expect(getInitialIndex('pointer', [])).toBe(-1)
  })

  it('is the item in force where the menu names one', () => {
    const three: MenuItem[] = [
      { id: 'one', text: 'One' },
      { id: 'two', text: 'Two' },
      { id: 'three', text: 'Three' },
    ]
    expect(getInitialIndex('keyboard', three, 'three')).toBe(2)
    expect(getInitialIndex('pointer', three, 'three')).toBe(-1)
  })

  it('is the first that can be chosen where the one in force is not among them', () => {
    expect(getInitialIndex('keyboard', items, 'gone')).toBe(1)
    expect(getInitialIndex('keyboard', items, 'one')).toBe(1)
  })
})

describe('the rules a menu draws between its groups', () => {
  const getRules = (items: readonly MenuItem[]) => groupItems(items).map((one) => one.isRule)

  it('remains where one group gives way to the next', () => {
    expect(
      getRules([
        { id: 'open', text: 'Open', group: 'open' },
        { id: 'note', text: 'New note', group: 'file' },
        { id: 'folder', text: 'New folder', group: 'file' },
        { id: 'remove', text: 'Remove', group: 'gone' },
      ]),
    ).toStrictEqual([false, true, false, true])
  })

  it('remains nowhere in a menu whose items name no group', () => {
    expect(
      getRules([
        { id: 'open', text: 'Open' },
        { id: 'copy', text: 'Copy path' },
      ]),
    ).toStrictEqual([false, false])
  })

  it('never remains above the first item, whatever group it names', () => {
    expect(getRules([{ id: 'remove', text: 'Remove', group: 'gone' }])).toStrictEqual([false])
  })

  it('remains again where a group arrives back after another', () => {
    expect(
      getRules([
        { id: 'one', text: 'One', group: 'file' },
        { id: 'two', text: 'Two', group: 'plex' },
        { id: 'three', text: 'Three', group: 'file' },
      ]),
    ).toStrictEqual([false, true, true])
  })

  it('carries every item through, in the order it was given', () => {
    const items: MenuItem[] = [
      { id: 'one', text: 'One', group: 'file', disabled: true },
      { id: 'two', text: 'Two', group: 'plex' },
    ]
    expect(groupItems(items).map((one) => one.id)).toStrictEqual(['one', 'two'])
    expect(groupItems(items)[0]?.disabled).toBe(true)
  })

  it('has nothing to draw for a menu holding nothing', () => {
    expect(groupItems([])).toStrictEqual([])
  })
})

describe('which items a search names', () => {
  const models: MenuItem[] = [
    { id: 'a', text: 'Opus 4', group: 'Claude Code' },
    { id: 'b', text: 'gpt-5 mini', detail: 'openai/gpt-5-mini', group: 'Numen Agent' },
    { id: 'c', text: 'Sonnet', group: 'Claude Code' },
  ]

  it('names every item where there are no words', () => {
    expect(filterItems(models, '')).toStrictEqual(models)
    expect(filterItems(models, '   ')).toStrictEqual(models)
  })

  it('finds a word in the text, in any case', () => {
    expect(filterItems(models, 'OPUS').map((one) => one.id)).toStrictEqual(['a'])
  })

  it('finds a word in the detail and in the group', () => {
    expect(filterItems(models, 'openai').map((one) => one.id)).toStrictEqual(['b'])
    expect(filterItems(models, 'claude').map((one) => one.id)).toStrictEqual(['a', 'c'])
  })

  it('wants every word found', () => {
    expect(filterItems(models, 'claude son').map((one) => one.id)).toStrictEqual(['c'])
    expect(filterItems(models, 'claude gpt')).toStrictEqual([])
  })
})
