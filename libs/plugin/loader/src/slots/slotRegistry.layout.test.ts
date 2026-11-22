/** What the plugin that draws a slot keeps of its entries. Every plugin here is invented. */
import { nextTick, shallowRef } from 'vue'
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
      getPoints: () => [{ pluginId: DRAWER, slot: SLOT_NAME, title: 'Below', menu: false }],
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

describe('the layout of a slot of commands', () => {
  const BUTTONS = 'titleBar.commands'
  const buttons: readonly DeclaredSlotEntry[] = [
    { pluginId: 'org.example.a', slot: BUTTONS, command: 'one', order: 1 },
    { pluginId: 'org.example.b', slot: BUTTONS, command: 'two', order: 2 },
  ]
  const readButtons = (layout: SlotLayout) =>
    createSlotRegistry({
      getEntries: () => buttons,
      drawers: {
        getPoints: () => [{ pluginId: DRAWER, slot: BUTTONS, title: 'Buttons', menu: false }],
        getLayout: () => layout,
      },
      findCommand: (command) => ({ title: command }),
      findView: () => undefined,
    })
      .getEntries(BUTTONS)
      .map((one) => (one.kind === 'command' ? one.command : one.kind))

  it('orders the commands by the list', () => {
    const layout = { [BUTTONS]: [{ id: 'org.example.b/command/org.example.b:two' }] }
    expect(readButtons(layout)).toStrictEqual(['org.example.b:two', 'org.example.a:one'])
  })

  it('leaves out a hidden command', () => {
    const layout = { [BUTTONS]: [{ id: 'org.example.a/command/org.example.a:one', hidden: true }] }
    expect(readButtons(layout)).toStrictEqual(['org.example.b:two'])
  })
})

describe('the layout of a slot while it is watched', () => {
  it('takes a hidden entry out and brings it back as the layout changes', async () => {
    const layout = shallowRef<SlotLayout | undefined>(undefined)
    const registry = createSlotRegistry({
      getEntries: () => entries,
      drawers: {
        getPoints: () => [{ pluginId: DRAWER, slot: SLOT_NAME, title: 'Below' }],
        getLayout: (pluginId) => (pluginId === DRAWER ? layout.value : undefined),
      },
      findCommand: () => undefined,
      findView: (pluginId, view) => ({ name: `${pluginId}-${view}` }),
    })
    const seen: string[][] = []
    registry.watchEntries(SLOT_NAME, (items) =>
      seen.push(items.map((one) => (one.kind === 'view' ? one.view : one.kind))),
    )

    layout.value = { [SLOT_NAME]: [{ id: KEY('b', 'two'), hidden: true }] }
    await nextTick()
    layout.value = { [SLOT_NAME]: [{ id: KEY('b', 'two') }] }
    await nextTick()

    expect(seen).toStrictEqual([
      ['one', 'two', 'three'],
      ['one', 'three'],
      ['two', 'one', 'three'],
    ])
  })
})

describe('the layout of a slot of commands in several groups', () => {
  const BUTTONS = 'titleBar.commands'
  const SEPARATOR = { id: '---' }
  const one = { id: 'org.example.a/command/org.example.a:one' }
  const two = { id: 'org.example.b/command/org.example.b:two' }
  const three = { id: 'org.example.c/command/org.example.c:three' }
  const grouped: readonly DeclaredSlotEntry[] = [
    { pluginId: 'org.example.a', slot: BUTTONS, command: 'one', group: '0-navigate' },
    { pluginId: 'org.example.b', slot: BUTTONS, command: 'two', group: '1-new' },
    { pluginId: 'org.example.c', slot: BUTTONS, command: 'three', group: '2-settings' },
  ]
  const readBar = (
    list: SlotLayout[string] | undefined,
    extra: readonly DeclaredSlotEntry[] = [],
  ) =>
    createSlotRegistry({
      getEntries: () => [...grouped, ...extra],
      drawers: {
        getPoints: () => [{ pluginId: DRAWER, slot: BUTTONS, title: 'Buttons', menu: false }],
        getLayout: () => (list ? { [BUTTONS]: list } : undefined),
      },
      findCommand: (command) => ({ title: command }),
      findView: () => undefined,
    })
      .getEntries(BUTTONS)
      .map((entry) => (entry.kind === 'command' ? entry.command.split(':')[1] : '|'))

  it('draws the groups with separators when nothing is stored', () => {
    expect(readBar(undefined)).toStrictEqual(['one', '|', 'two', '|', 'three'])
  })

  it('draws the entries and the separators in the order of the list', () => {
    expect(readBar([three, SEPARATOR, one, two])).toStrictEqual(['three', '|', 'one', 'two'])
    expect(readBar([one, SEPARATOR, two, SEPARATOR, three])).toStrictEqual([
      'one',
      '|',
      'two',
      '|',
      'three',
    ])
  })

  it('moves an entry into another group by where the list puts it', () => {
    expect(readBar([one, two, SEPARATOR, three])).toStrictEqual(['one', 'two', '|', 'three'])
  })

  it('draws no separator that opens the slot, closes it or follows another', () => {
    expect(readBar([SEPARATOR, one, SEPARATOR, SEPARATOR, two, three, SEPARATOR])).toStrictEqual([
      'one',
      '|',
      'two',
      'three',
    ])
  })

  it('leaves no separator for a group whose entries are all hidden', () => {
    expect(readBar([one, SEPARATOR, { ...two, hidden: true }, SEPARATOR, three])).toStrictEqual([
      'one',
      '|',
      'three',
    ])
  })

  it('puts an entry the list does not name after the last group, in its order', () => {
    const extra: DeclaredSlotEntry[] = [
      { pluginId: 'org.example.d', slot: BUTTONS, command: 'four', group: '0-navigate' },
    ]
    expect(readBar([one, SEPARATOR, two, three, SEPARATOR], extra)).toStrictEqual([
      'one',
      '|',
      'two',
      'three',
      'four',
    ])
  })

  it('ignores a repeated entry and an id no plugin places', () => {
    expect(readBar([two, { id: 'gone' }, two, one])).toStrictEqual(['two', 'one', 'three'])
  })
})
