/** What the plugin that draws a slot keeps of its entries. Every plugin here is invented. */
import { describe, expect, it } from 'vitest'
import { createSlotRegistry, type DeclaredSlotEntry } from './slotRegistry'
import type { SlotLayout } from './slotLayout'

const DRAWER = 'org.example.plex'
const SLOT_NAME = 'plexGraph.nodeBelow'
const entries: readonly DeclaredSlotEntry[] = [
  { pluginId: 'org.example.a', slot: SLOT_NAME, view: 'one', order: 1 },
  { pluginId: 'org.example.b', slot: SLOT_NAME, view: 'two', order: 2 },
  { pluginId: 'org.example.c', slot: SLOT_NAME, view: 'three', order: 3 },
]
const KEY = (plugin: string, view: string) => `org.example.${plugin}/view/${view}`

const read = (layout: SlotLayout | undefined) => {
  const registry = createSlotRegistry({
    getEntries: () => entries,
    drawers: {
      getPoints: () => [{ pluginId: DRAWER, slot: SLOT_NAME, title: 'Below' }],
      getLayout: (pluginId) => (pluginId === DRAWER ? layout : undefined),
    },
    findCommand: () => undefined,
    findView: (pluginId, view) => ({ name: `${pluginId}-${view}` }),
  })
  return registry.getEntries(SLOT_NAME).map((one) => (one.kind === 'view' ? one.view : one.kind))
}

describe('the layout of a slot', () => {
  it('keeps the order of the manifests when nothing is stored', () => {
    expect(read(undefined)).toStrictEqual(['one', 'two', 'three'])
  })

  it('draws the entries in the order of the list', () => {
    const layout = { [SLOT_NAME]: [{ id: KEY('c', 'three') }, { id: KEY('a', 'one') }] }
    expect(read(layout)).toStrictEqual(['three', 'one', 'two'])
  })

  it('leaves out the hidden entries', () => {
    const layout = { [SLOT_NAME]: [{ id: KEY('b', 'two'), hidden: true }] }
    expect(read(layout)).toStrictEqual(['one', 'three'])
  })

  it('ignores ids no plugin places', () => {
    const layout = { [SLOT_NAME]: [{ id: KEY('gone', 'view') }, { id: KEY('b', 'two') }] }
    expect(read(layout)).toStrictEqual(['two', 'one', 'three'])
  })

  it('ignores the layout of another slot', () => {
    expect(read({ 'plexGraph.node': [{ id: KEY('a', 'one'), hidden: true }] })).toStrictEqual([
      'one',
      'two',
      'three',
    ])
  })
})
