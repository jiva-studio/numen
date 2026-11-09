/** The layout of the slots: its keys, how a settings value is read, and what a drop writes. Every plugin here is invented. */
import { describe, expect, it } from 'vitest'
import type { SlotName } from '@numen/plugin-sdk'
import { getLayoutKey, moveSlotEntry, readSlotLayout, type SlotLayout } from './slotLayout'

const STATS = 'titleBar.status|org.example.reef/stats'
const TASKS = 'titleBar.status|org.example.tide/status'
const UPDATER = 'titleBar.status|org.example.harbor/update'

const SLOT_NAMES: readonly SlotName[] = [
  'titleBar',
  'statusBar',
  'titleBar.commands',
  'titleBar.status',
  'statusBar.left',
  'statusBar.right',
  'fileTree.context',
  'tabStrip.context',
  'plexGraph.context',
  'palette.searchResult',
]

const drawnAs = (slots: Partial<Record<SlotName, readonly string[]>>) =>
  ({
    ...Object.fromEntries(SLOT_NAMES.map((name) => [name, []])),
    ...slots,
  }) as Record<SlotName, readonly string[]>

describe('the layout key of an entry', () => {
  it('is the home slot, the plugin and the command as the manifest writes it', () => {
    expect(
      getLayoutKey({
        pluginId: 'org.example.tide',
        slot: 'titleBar.commands',
        command: 'tide.open',
      }),
    ).toBe('titleBar.commands|org.example.tide/tide.open')
    expect(
      getLayoutKey({
        pluginId: 'org.example.ledger',
        slot: 'titleBar.commands',
        command: 'app:note.create',
      }),
    ).toBe('titleBar.commands|org.example.ledger/app:note.create')
  })

  it('is the home slot, the plugin and the view id for a view', () => {
    expect(
      getLayoutKey({ pluginId: 'org.example.tide', slot: 'titleBar.status', view: 'status' }),
    ).toBe(TASKS)
  })

  it('names the application as app for its own entries', () => {
    expect(
      getLayoutKey({ pluginId: 'app', slot: 'titleBar.commands', command: 'app:settings.open' }),
    ).toBe('titleBar.commands|app/app:settings.open')
  })

  it('differs for one command placed in two slots', () => {
    const bar = getLayoutKey({
      pluginId: 'org.example.tide',
      slot: 'titleBar.commands',
      command: 'tide.open',
    })
    const menu = getLayoutKey({
      pluginId: 'org.example.tide',
      slot: 'fileTree.context',
      command: 'tide.open',
    })

    expect(bar).toBe('titleBar.commands|org.example.tide/tide.open')
    expect(menu).toBe('fileTree.context|org.example.tide/tide.open')
  })

  it('is none for a submenu', () => {
    expect(
      getLayoutKey({
        pluginId: 'org.example.tide',
        slot: 'titleBar.commands',
        submenu: 'tide.more',
        title: 'More',
      }),
    ).toBeUndefined()
  })
})

describe('a layout read from the settings', () => {
  it('keeps the lists of the slots', () => {
    expect(
      readSlotLayout({ 'titleBar.status': [TASKS, STATS], 'statusBar.right': [UPDATER] }),
    ).toStrictEqual({ 'titleBar.status': [TASKS, STATS], 'statusBar.right': [UPDATER] })
  })

  it('keeps the list of a menu slot', () => {
    expect(readSlotLayout({ 'fileTree.context': [STATS] })).toStrictEqual({
      'fileTree.context': [STATS],
    })
  })

  it('leaves out a slot that slots.json does not declare and a list that is not a list of strings', () => {
    expect(
      readSlotLayout({
        'titleBar.status': [TASKS],
        'sideBar.top': [STATS],
        'statusBar.left': STATS,
        'statusBar.right': [UPDATER, 3],
      }),
    ).toStrictEqual({ 'titleBar.status': [TASKS] })
  })

  it('is empty for no value and for a value that is no object', () => {
    for (const value of [undefined, null, 'titleBar.status', [TASKS]]) {
      expect(readSlotLayout(value)).toStrictEqual({})
    }
  })
})

describe('a drop of a slot entry', () => {
  it('within its slot writes that slot alone, in the new order', () => {
    const change = moveSlotEntry({}, drawnAs({ 'titleBar.status': [STATS, TASKS] }), {
      key: TASKS,
      from: 'titleBar.status',
      to: 'titleBar.status',
      before: STATS,
    })

    expect(change).toStrictEqual({ 'titleBar.status': [TASKS, STATS] })
  })

  it('into another slot writes both slots', () => {
    const change = moveSlotEntry({}, drawnAs({ 'titleBar.status': [STATS, UPDATER, TASKS] }), {
      key: UPDATER,
      from: 'titleBar.status',
      to: 'statusBar.right',
    })

    expect(change).toStrictEqual({
      'titleBar.status': [STATS, TASKS],
      'statusBar.right': [UPDATER],
    })
  })

  it('into a menu slot writes both slots', () => {
    const change = moveSlotEntry({}, drawnAs({ 'titleBar.status': [STATS, UPDATER] }), {
      key: UPDATER,
      from: 'titleBar.status',
      to: 'fileTree.context',
    })

    expect(change).toStrictEqual({
      'titleBar.status': [STATS],
      'fileTree.context': [UPDATER],
    })
  })

  it('into another slot goes before the key it is dropped on', () => {
    const change = moveSlotEntry(
      {},
      drawnAs({ 'titleBar.status': [STATS, UPDATER], 'statusBar.right': [TASKS] }),
      { key: UPDATER, from: 'titleBar.status', to: 'statusBar.right', before: TASKS },
    )

    expect(change).toStrictEqual({
      'titleBar.status': [STATS],
      'statusBar.right': [UPDATER, TASKS],
    })
  })

  it('keeps a stored key that is not drawn right after the key that preceded it', () => {
    const layout: SlotLayout = {
      'titleBar.commands': [
        'titleBar.commands|a/one',
        'titleBar.commands|x/gone',
        'titleBar.commands|b/two',
      ],
    }

    const change = moveSlotEntry(
      layout,
      drawnAs({ 'titleBar.commands': ['titleBar.commands|a/one', 'titleBar.commands|b/two'] }),
      {
        key: 'titleBar.commands|b/two',
        from: 'titleBar.commands',
        to: 'titleBar.commands',
        before: 'titleBar.commands|a/one',
      },
    )

    expect(change).toStrictEqual({
      'titleBar.commands': [
        'titleBar.commands|b/two',
        'titleBar.commands|a/one',
        'titleBar.commands|x/gone',
      ],
    })
  })

  it('keeps a stored key that is not drawn first when no stored key preceded it', () => {
    const layout: SlotLayout = {
      'titleBar.commands': [
        'titleBar.commands|x/gone',
        'titleBar.commands|a/one',
        'titleBar.commands|b/two',
      ],
    }

    const change = moveSlotEntry(
      layout,
      drawnAs({ 'titleBar.commands': ['titleBar.commands|a/one', 'titleBar.commands|b/two'] }),
      { key: 'titleBar.commands|a/one', from: 'titleBar.commands', to: 'titleBar.commands' },
    )

    expect(change).toStrictEqual({
      'titleBar.commands': [
        'titleBar.commands|x/gone',
        'titleBar.commands|b/two',
        'titleBar.commands|a/one',
      ],
    })
  })

  it('writes every drawn key of the slot, so entries never placed join the user order', () => {
    const layout: SlotLayout = { 'titleBar.status': [TASKS] }

    const change = moveSlotEntry(layout, drawnAs({ 'titleBar.status': [TASKS, STATS, UPDATER] }), {
      key: UPDATER,
      from: 'titleBar.status',
      to: 'titleBar.status',
      before: TASKS,
    })

    expect(change).toStrictEqual({ 'titleBar.status': [UPDATER, TASKS, STATS] })
  })

  it('before the key that already follows it changes nothing', () => {
    const drawn = drawnAs({ 'titleBar.status': [STATS, TASKS, UPDATER] })

    expect(
      moveSlotEntry({}, drawn, {
        key: STATS,
        from: 'titleBar.status',
        to: 'titleBar.status',
        before: TASKS,
      }),
    ).toStrictEqual({})
    expect(
      moveSlotEntry({ 'titleBar.status': [STATS, TASKS, UPDATER] }, drawn, {
        key: TASKS,
        from: 'titleBar.status',
        to: 'titleBar.status',
        before: UPDATER,
      }),
    ).toStrictEqual({})
  })

  it('at the end of its own slot when it is last changes nothing', () => {
    expect(
      moveSlotEntry({}, drawnAs({ 'statusBar.left': [STATS, TASKS] }), {
        key: TASKS,
        from: 'statusBar.left',
        to: 'statusBar.left',
      }),
    ).toStrictEqual({})
  })
})
