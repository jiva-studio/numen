/** The entries of the slots of a window, as their owners read them. Every plugin here is invented. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, shallowRef } from 'vue'
import type { CommandTarget, SlotItem } from '@numen/plugin-sdk'
import { createSlotRegistry, type DeclaredSlotEntry, type SlotRegistryDeps } from './slotRegistry'

const LEDGER = 'org.example.ledger'
const HARBOR = 'org.example.harbor'
const TIDE = 'org.example.tide'

const TITLES: Record<string, string> = {
  [`${LEDGER}:ledger.open`]: 'Open ledger',
  [`${LEDGER}:ledger.sum`]: 'Sum',
  [`${HARBOR}:harbor.open`]: 'Open harbor',
  [`${HARBOR}:harbor.note`]: 'Harbor note',
  [`${TIDE}:tide.open`]: 'Open tide',
  'app:settings.open': 'Open settings',
}

const NOTE: CommandTarget = {
  path: 'notes/Bees.md',
  title: 'Bees',
  isFolder: false,
  others: [],
  tab: '',
}

function setup(entries: readonly DeclaredSlotEntry[], overrides: Partial<SlotRegistryDeps> = {}) {
  const list = shallowRef<readonly DeclaredSlotEntry[] | undefined>(entries)
  const reportIssue = vi.fn()
  const registry = createSlotRegistry({
    getEntries: () => list.value,
    findCommand: (command, target) => {
      const title = TITLES[command]
      if (title === undefined) return undefined
      // A command over a note is left out where the slot has no note.
      if (command.endsWith('.note') && !target?.path) return undefined
      return { title }
    },
    findView: (pluginId, view) =>
      view === 'balance' ? { name: `${pluginId}-${view}` } : undefined,
    reportIssue,
    ...overrides,
  })
  return { registry, list, reportIssue }
}

const describeItem = (item: SlotItem): string => {
  if (item.kind === 'separator') return '|'
  if (item.kind === 'command') return item.command
  if (item.kind === 'view') return `view ${item.view}`
  return `submenu ${item.submenu} [${item.entries.map(describeItem).join(', ')}]`
}

const describeSlot = (items: readonly SlotItem[]) => items.map(describeItem)

describe('the entries of a slot', () => {
  it('are sorted by group, then order, then plugin id, with a separator between groups', () => {
    const { registry } = setup([
      { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.open', group: '1-go', order: 2 },
      {
        pluginId: LEDGER,
        slot: 'titleBar.commands',
        command: 'ledger.open',
        group: '1-go',
        order: 2,
      },
      {
        pluginId: HARBOR,
        slot: 'titleBar.commands',
        command: 'harbor.open',
        group: '1-go',
        order: 1,
      },
      { pluginId: 'app', slot: 'titleBar.commands', command: 'app:settings.open' },
      { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.sum', group: '0-make' },
    ])

    expect(describeSlot(registry.getEntries('titleBar.commands'))).toStrictEqual([
      `${LEDGER}:ledger.sum`,
      '|',
      `${HARBOR}:harbor.open`,
      `${LEDGER}:ledger.open`,
      `${TIDE}:tide.open`,
      '|',
      'app:settings.open',
    ])
  })

  it('put an entry without an order after the ordered ones of its group', () => {
    const { registry } = setup([
      { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open' },
      { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open', order: 5 },
    ])

    expect(describeSlot(registry.getEntries('titleBar.commands'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
      `${HARBOR}:harbor.open`,
    ])
  })

  it('leave out a command unavailable over the target, and the separator of a group left empty', () => {
    const { registry } = setup([
      { pluginId: HARBOR, slot: 'fileTree.context', command: 'harbor.note', group: '0-note' },
      { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.open', group: '1-open' },
      { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.gone', group: '2-gone' },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
    expect(describeSlot(registry.getEntries('fileTree.context', NOTE))).toStrictEqual([
      `${HARBOR}:harbor.note`,
      '|',
      `${LEDGER}:ledger.open`,
    ])
  })

  it('carry a key of the plugin and what it places, and the look of the command', () => {
    const { registry } = setup([
      { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open' },
      { pluginId: LEDGER, slot: 'titleBar.status', view: 'balance' },
    ])

    expect(registry.getEntries('titleBar.commands')).toStrictEqual([
      {
        kind: 'command',
        key: `${LEDGER}/command/${LEDGER}:ledger.open`,
        pluginId: LEDGER,
        command: `${LEDGER}:ledger.open`,
        title: 'Open ledger',
        shortcut: '',
        hasItems: false,
      },
    ])
    expect(registry.getEntries('titleBar.status')).toStrictEqual([
      {
        kind: 'view',
        key: `${LEDGER}/view/balance`,
        pluginId: LEDGER,
        view: 'balance',
        component: { name: `${LEDGER}-balance` },
      },
    ])
  })

  it('mark a command that lists items, which the owner draws as a submenu of them', () => {
    const { registry } = setup(
      [{ pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.sum' }],
      {
        findCommand: () => ({ title: 'Sum', hasItems: true }),
      },
    )

    expect(registry.getEntries('fileTree.context', NOTE)).toMatchObject([
      { kind: 'command', command: `${LEDGER}:ledger.sum`, hasItems: true },
    ])
  })

  it('carry the key bound to the command with the values of the entry, one key for each entry', () => {
    const keys: Record<string, string> = {
      [`${LEDGER}:ledger.open?kind=plex`]: 'mod+shift+p',
      [`${LEDGER}:ledger.open?kind=files`]: 'mod+alt+e',
    }
    const { registry } = setup(
      [
        {
          pluginId: LEDGER,
          slot: 'titleBar.commands',
          command: 'ledger.open',
          values: { kind: 'plex' },
        },
        {
          pluginId: LEDGER,
          slot: 'titleBar.commands',
          command: 'ledger.open',
          values: { kind: 'files' },
        },
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open' },
      ],
      { getKeyFor: (command, values) => keys[`${command}?kind=${values?.kind}`] },
    )

    expect(
      registry
        .getEntries('titleBar.commands')
        .map((one) => (one.kind === 'command' ? one.shortcut : '')),
    ).toStrictEqual(['', 'mod+alt+e', 'mod+shift+p'])
  })

  it('leave out a view that has not been registered', () => {
    const { registry } = setup([{ pluginId: LEDGER, slot: 'titleBar.status', view: 'chart' }])

    expect(registry.getEntries('titleBar.status')).toStrictEqual([])
  })

  it('are none until the first list of entries has loaded', () => {
    const { registry, list } = setup([])
    list.value = undefined

    expect(registry.getEntries('titleBar.commands')).toStrictEqual([])
  })
})

describe('a submenu', () => {
  it('holds the entries of every plugin that names it, by the same rules', () => {
    const { registry } = setup([
      {
        pluginId: LEDGER,
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New',
        group: '0-new',
      },
      { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.open', group: '1-open' },
      { pluginId: HARBOR, slot: 'ledger.new', command: 'harbor.open' },
      { pluginId: LEDGER, slot: 'ledger.new', command: 'ledger.sum', order: 1 },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context'))).toStrictEqual([
      `submenu ledger.new [${LEDGER}:ledger.sum, ${HARBOR}:harbor.open]`,
      '|',
      `${LEDGER}:ledger.open`,
    ])
    const [submenu] = registry.getEntries('fileTree.context')
    expect(submenu).toMatchObject({
      kind: 'submenu',
      key: `${LEDGER}/submenu/ledger.new`,
      title: 'New',
    })
  })

  it('is left out while every entry in it is hidden, with its separator', () => {
    const { registry } = setup([
      {
        pluginId: LEDGER,
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New',
        group: '0-new',
      },
      { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.open', group: '1-open' },
      { pluginId: HARBOR, slot: 'ledger.new', command: 'harbor.note' },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
  })
})

describe('the condition of an entry', () => {
  const FOLDER: CommandTarget = {
    path: 'notes',
    title: 'notes',
    isFolder: true,
    others: [],
    tab: '',
  }
  const ROOT: CommandTarget = { path: '', title: '', isFolder: true, others: [], tab: '' }

  it('leaves the entry out over a target where it is false', () => {
    const { registry } = setup([
      {
        pluginId: LEDGER,
        slot: 'fileTree.context',
        command: 'ledger.open',
        when: 'target.isFolder && !target.isRoot',
      },
      { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.sum' },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context', FOLDER))).toStrictEqual([
      `${LEDGER}:ledger.open`,
      `${LEDGER}:ledger.sum`,
    ])
    expect(describeSlot(registry.getEntries('fileTree.context', ROOT))).toStrictEqual([
      `${LEDGER}:ledger.sum`,
    ])
    expect(describeSlot(registry.getEntries('fileTree.context', NOTE))).toStrictEqual([
      `${LEDGER}:ledger.sum`,
    ])
  })

  it('leaves out a submenu and its separator where it is false', () => {
    const { registry } = setup([
      { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.sum', group: '0' },
      {
        pluginId: LEDGER,
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New',
        group: '1',
        when: 'target.isMultiple',
      },
      { pluginId: LEDGER, slot: 'ledger.new', command: 'ledger.open' },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context', FOLDER))).toStrictEqual([
      `${LEDGER}:ledger.sum`,
    ])
  })

  it('that is malformed leaves the entry out and is reported as an error', () => {
    const { registry, reportIssue } = setup([
      {
        pluginId: LEDGER,
        slot: 'fileTree.context',
        command: 'ledger.open',
        when: 'target.isRemote',
      },
    ])

    expect(registry.getEntries('fileTree.context', FOLDER)).toStrictEqual([])
    expect(reportIssue).toHaveBeenCalledWith(
      LEDGER,
      'slots: the condition "target.isRemote" of "ledger.open" cannot be read: "target.isRemote" reads target.isRemote, which is no variable of when',
      'error',
    )
  })
})

describe('the diagnostics of the entries', () => {
  it('warn of an entry in a place no plugin declares', () => {
    const { registry, reportIssue } = setup([
      { pluginId: HARBOR, slot: 'ledger.new', command: 'harbor.open' },
    ])

    registry.getEntries('fileTree.context')

    expect(reportIssue).toHaveBeenCalledWith(
      HARBOR,
      'slots: "harbor.open" is placed in "ledger.new", which is neither a slot of the window nor a submenu a plugin declares',
      'warning',
    )
  })

  it('give the second plugin to declare a submenu an error, and keep the submenu of the first', () => {
    const { registry, reportIssue } = setup([
      { pluginId: TIDE, slot: 'fileTree.context', submenu: 'ledger.new', title: 'Tide new' },
      { pluginId: LEDGER, slot: 'fileTree.context', submenu: 'ledger.new', title: 'New' },
      { pluginId: HARBOR, slot: 'ledger.new', command: 'harbor.open' },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context'))).toStrictEqual([
      `submenu ledger.new [${HARBOR}:harbor.open]`,
    ])
    expect(registry.getEntries('fileTree.context')[0]).toMatchObject({ pluginId: LEDGER })
    expect(reportIssue).toHaveBeenCalledWith(
      TIDE,
      `slots: the submenu "ledger.new" is declared by ${LEDGER} already`,
      'error',
    )
  })

  it('report a cycle of submenus as an error, and draw the submenus of the cycle nowhere', () => {
    const { registry, reportIssue } = setup([
      { pluginId: LEDGER, slot: 'fileTree.context', submenu: 'ledger.outer', title: 'Outer' },
      { pluginId: LEDGER, slot: 'ledger.outer', submenu: 'ledger.inner', title: 'Inner' },
      { pluginId: HARBOR, slot: 'ledger.inner', submenu: 'harbor.back', title: 'Back' },
      { pluginId: HARBOR, slot: 'harbor.back', submenu: 'harbor.loop', title: 'Loop' },
      { pluginId: LEDGER, slot: 'harbor.loop', submenu: 'ledger.inner2', title: 'Loop' },
      { pluginId: HARBOR, slot: 'ledger.inner2', submenu: 'harbor.back2', title: 'Loop' },
      { pluginId: HARBOR, slot: 'harbor.back2', command: 'harbor.open' },
      { pluginId: HARBOR, slot: 'ledger.inner', command: 'harbor.open' },
      { pluginId: TIDE, slot: 'tide.a', submenu: 'tide.b', title: 'B' },
      { pluginId: TIDE, slot: 'tide.b', submenu: 'tide.a', title: 'A' },
      { pluginId: TIDE, slot: 'tide.a', command: 'tide.open' },
    ])

    expect(describeSlot(registry.getEntries('fileTree.context'))).toStrictEqual([
      `submenu ledger.outer [submenu ledger.inner [${HARBOR}:harbor.open, submenu harbor.back [submenu harbor.loop [submenu ledger.inner2 [submenu harbor.back2 [${HARBOR}:harbor.open]]]]]]`,
    ])
    expect(reportIssue).toHaveBeenCalledWith(
      TIDE,
      'slots: the submenu "tide.a" holds itself through tide.a, tide.b',
      'error',
    )
  })

  it('report each issue once, however often the entries are read', () => {
    const { registry, reportIssue } = setup([
      { pluginId: HARBOR, slot: 'ledger.new', command: 'harbor.open' },
    ])

    registry.getEntries('fileTree.context')
    registry.getEntries('titleBar.commands')

    expect(reportIssue).toHaveBeenCalledTimes(1)
  })
})

describe('a watcher of a slot', () => {
  it('receives the entries at once and again when they change', async () => {
    const { registry, list } = setup([
      { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open' },
    ])
    const handler = vi.fn()

    const watching = registry.watchEntries('titleBar.commands', handler)
    list.value = [
      ...(list.value ?? []),
      { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open' },
    ]
    await nextTick()
    watching.dispose()
    list.value = []
    await nextTick()

    expect(handler.mock.calls.map(([items]) => describeSlot(items as SlotItem[]))).toStrictEqual([
      [`${LEDGER}:ledger.open`],
      [`${HARBOR}:harbor.open`, `${LEDGER}:ledger.open`],
    ])
  })
})

describe('the values of an entry', () => {
  it('stay on the command item it draws', () => {
    const { registry } = setup([
      {
        pluginId: LEDGER,
        slot: 'titleBar.commands',
        command: 'ledger.open',
        values: { book: 'main' },
      },
      { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open' },
    ])

    const items = registry.getEntries('titleBar.commands')

    expect(
      items.find((one) => one.key === `${LEDGER}/command/${LEDGER}:ledger.open?book=main`),
    ).toMatchObject({
      values: { book: 'main' },
    })
    expect(
      items.find((one) => one.key === `${HARBOR}/command/${HARBOR}:harbor.open`),
    ).not.toHaveProperty('values')
  })
})

describe('the entries of one command with different values', () => {
  it('are drawn as items with different keys, and the same values draw one item', () => {
    const { registry } = setup([
      {
        pluginId: LEDGER,
        slot: 'titleBar.commands',
        command: 'ledger.open',
        values: { book: 'main' },
        order: 1,
      },
      {
        pluginId: LEDGER,
        slot: 'titleBar.commands',
        command: 'ledger.open',
        values: { book: 'spare' },
        order: 2,
      },
      {
        pluginId: LEDGER,
        slot: 'titleBar.commands',
        command: 'ledger.open',
        values: { book: 'spare' },
        order: 3,
      },
    ])

    const items = registry.getEntries('titleBar.commands')

    expect(items.map((one) => one.key)).toStrictEqual([
      `${LEDGER}/command/${LEDGER}:ledger.open?book=main`,
      `${LEDGER}/command/${LEDGER}:ledger.open?book=spare`,
    ])
  })
})

describe('the look of a command entry', () => {
  it('is the title and icon of the entry where it has them, and the look of the command otherwise', () => {
    const icon = { name: 'pin' }
    const { registry } = setup([
      {
        pluginId: LEDGER,
        slot: 'titleBar.commands',
        command: 'ledger.open',
        title: 'Pinned',
        icon,
      },
      { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open' },
    ])

    const [harbor, ledger] = registry.getEntries('titleBar.commands')

    expect(ledger).toMatchObject({ title: 'Pinned', icon })
    expect(harbor).toMatchObject({ title: 'Open harbor' })
  })
})
