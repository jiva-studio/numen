/** The entries of the slots of a window, as their owners read them. Every plugin here is invented. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, shallowRef } from 'vue'
import type { CommandTarget, SlotItem, SlotName } from '@numen/plugin-sdk'
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
      return { title, shortcut: '' }
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
        findCommand: () => ({ title: 'Sum', shortcut: '', hasItems: true }),
      },
    )

    expect(registry.getEntries('fileTree.context', NOTE)).toMatchObject([
      { kind: 'command', command: `${LEDGER}:ledger.sum`, hasItems: true },
    ])
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

describe('the layout of the slots', () => {
  const REEF = 'org.example.reef'
  const STATS: DeclaredSlotEntry = { pluginId: REEF, slot: 'titleBar.status', view: 'stats' }
  const TASKS: DeclaredSlotEntry = { pluginId: TIDE, slot: 'titleBar.status', view: 'status' }
  const UPDATER: DeclaredSlotEntry = { pluginId: HARBOR, slot: 'titleBar.status', view: 'update' }
  const STATS_KEY = `titleBar.status|${REEF}/stats`
  const TASKS_KEY = `titleBar.status|${TIDE}/status`
  const UPDATER_KEY = `titleBar.status|${HARBOR}/update`

  const setupWithLayout = (
    entries: readonly DeclaredSlotEntry[],
    layout: Record<string, readonly string[]> | undefined,
    overrides: Partial<SlotRegistryDeps> = {},
  ) => {
    const current = shallowRef(layout)
    const deps = {
      findView: (pluginId: string, view: string) => ({ name: `${pluginId}-${view}` }),
      getLayout: () => current.value,
      ...overrides,
    }
    return { ...setup(entries, deps as Partial<SlotRegistryDeps>), layout: current }
  }

  it('draws listed entries in the listed order', () => {
    const { registry } = setupWithLayout([STATS, TASKS], {
      'titleBar.status': [TASKS_KEY, STATS_KEY],
    })

    expect(describeSlot(registry.getEntries('titleBar.status'))).toStrictEqual([
      'view status',
      'view stats',
    ])
  })

  it('draws a listed entry in the listed slot and not in the slot of its manifest', () => {
    const { registry } = setupWithLayout([STATS, UPDATER], {
      'statusBar.right': [UPDATER_KEY],
    })

    expect(describeSlot(registry.getEntries('titleBar.status'))).toStrictEqual(['view stats'])
    expect(describeSlot(registry.getEntries('statusBar.right'))).toStrictEqual(['view update'])
  })

  it('draws the unlisted entries after the listed ones, by group, order and plugin id', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open', order: 1 },
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open', order: 2 },
        { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.open', order: 3 },
      ],
      { 'titleBar.commands': [`titleBar.commands|${TIDE}/tide.open`] },
    )

    expect(describeSlot(registry.getEntries('titleBar.commands'))).toStrictEqual([
      `${TIDE}:tide.open`,
      `${HARBOR}:harbor.open`,
      `${LEDGER}:ledger.open`,
    ])
  })

  it('without a layout draws the order and the separators of the manifests, each separator with its own key', () => {
    const entries: DeclaredSlotEntry[] = [
      { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.open', group: '1-go', order: 2 },
      { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open', group: '1-go' },
      { pluginId: 'app', slot: 'titleBar.commands', command: 'app:settings.open' },
      { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.sum', group: '0-make' },
    ]

    for (const layout of [undefined, {}]) {
      const { registry } = setupWithLayout(entries, layout)
      const items = registry.getEntries('titleBar.commands')

      expect(describeSlot(items)).toStrictEqual([
        `${LEDGER}:ledger.sum`,
        '|',
        `${TIDE}:tide.open`,
        `${HARBOR}:harbor.open`,
        '|',
        'app:settings.open',
      ])
      expect(items.filter((one) => one.kind === 'separator').map((one) => one.key)).toStrictEqual([
        'titleBar.commands/separator/0',
        'titleBar.commands/separator/1',
      ])
    }
  })

  it('puts a separator between listed neighbours of different groups, never first, last or doubled', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: HARBOR, slot: 'titleBar.commands', command: 'harbor.open', group: 'navigate' },
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.sum', group: 'new' },
        { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.open', group: 'navigate' },
      ],
      {
        'titleBar.commands': [
          `titleBar.commands|${HARBOR}/harbor.open`,
          `titleBar.commands|${LEDGER}/ledger.sum`,
          `titleBar.commands|${TIDE}/tide.open`,
        ],
      },
    )
    const items = registry.getEntries('titleBar.commands')

    expect(describeSlot(items)).toStrictEqual([
      `${HARBOR}:harbor.open`,
      '|',
      `${LEDGER}:ledger.sum`,
      '|',
      `${TIDE}:tide.open`,
    ])
    const keys = items.map((one) => one.key)
    expect(new Set(keys).size).toBe(keys.length)
  })

  it('skips listed keys that draw nothing and draws the rest in the listed order', () => {
    const { registry } = setupWithLayout(
      [
        STATS,
        TASKS,
        { pluginId: LEDGER, slot: 'titleBar.status', view: 'balance', when: 'target.isFolder' },
      ],
      {
        'titleBar.status': [
          'titleBar.status|org.example.disabled/clock',
          TASKS_KEY,
          'titleBar.status|org.example.missing/weather',
          `titleBar.status|${LEDGER}/balance`,
          STATS_KEY,
        ],
      },
    )

    expect(describeSlot(registry.getEntries('titleBar.status'))).toStrictEqual([
      'view status',
      'view stats',
    ])
  })

  it('draws a key listed in two slots in the first of them in the order of slots.json, and a key listed twice once at its first place', () => {
    const left = (pluginId: string, view: string): DeclaredSlotEntry => ({
      pluginId,
      slot: 'statusBar.left',
      view,
    })
    const { registry } = setupWithLayout([UPDATER, left(LEDGER, 'words'), left(REEF, 'clock')], {
      'statusBar.right': [
        `statusBar.left|${LEDGER}/words`,
        `statusBar.left|${REEF}/clock`,
        `statusBar.left|${LEDGER}/words`,
        UPDATER_KEY,
      ],
      'titleBar.status': [UPDATER_KEY],
    })

    expect(describeSlot(registry.getEntries('titleBar.status'))).toStrictEqual(['view update'])
    expect(describeSlot(registry.getEntries('statusBar.right'))).toStrictEqual([
      'view words',
      'view clock',
    ])
    expect(describeSlot(registry.getEntries('statusBar.left'))).toStrictEqual([])
  })

  it('takes a menu slot into the layout, by the key of the menu placement', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open' },
        { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.open' },
      ],
      { 'statusBar.left': [`fileTree.context|${LEDGER}/ledger.open`] },
    )

    expect(describeSlot(registry.getEntries('fileTree.context', NOTE))).toStrictEqual([])
    expect(describeSlot(registry.getEntries('titleBar.commands'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
    expect(describeSlot(registry.getEntries('statusBar.left'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
  })

  it('draws a menu slot as the manifests place it when the layout does not list it', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open' },
        { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.open' },
      ],
      { 'titleBar.commands': [`titleBar.commands|${LEDGER}/ledger.open`] },
    )

    expect(describeSlot(registry.getEntries('fileTree.context', NOTE))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
  })

  it('keeps the placements of one command in two slots apart, each with its own key', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open' },
        { pluginId: LEDGER, slot: 'statusBar.left', command: 'ledger.open' },
      ],
      { 'statusBar.right': [`statusBar.left|${LEDGER}/ledger.open`] },
    )

    expect(describeSlot(registry.getEntries('titleBar.commands'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
    expect(describeSlot(registry.getEntries('statusBar.left'))).toStrictEqual([])
    expect(describeSlot(registry.getEntries('statusBar.right'))).toStrictEqual([
      `${LEDGER}:ledger.open`,
    ])
  })

  it('redraws a watched slot when the layout changes', async () => {
    const { registry, layout } = setupWithLayout([STATS, TASKS], undefined)
    const handler = vi.fn()

    const watching = registry.watchEntries('titleBar.status', handler)
    layout.value = { 'titleBar.status': [TASKS_KEY] }
    await nextTick()
    watching.dispose()

    expect(handler.mock.calls.map(([items]) => describeSlot(items as SlotItem[]))).toStrictEqual([
      ['view stats', 'view status'],
      ['view status', 'view stats'],
    ])
  })

  it('finds the layout key of a drawn command or view, and none for a separator or an unknown item', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open', group: 'a' },
        { pluginId: 'app', slot: 'titleBar.commands', command: 'app:settings.open', group: 'b' },
        TASKS,
      ],
      undefined,
    )

    expect(registry.findLayoutKey(`${LEDGER}/command/${LEDGER}:ledger.open`)).toBe(
      `titleBar.commands|${LEDGER}/ledger.open`,
    )
    expect(registry.findLayoutKey('app/command/app:settings.open')).toBe(
      'titleBar.commands|app/app:settings.open',
    )
    expect(registry.findLayoutKey(`${TIDE}/view/status`)).toBe(TASKS_KEY)
    expect(registry.findLayoutKey('titleBar.commands/separator/0')).toBeUndefined()
    expect(registry.findLayoutKey(`${REEF}/view/unknown`)).toBeUndefined()
  })

  it('lists the layout keys a slot draws, in drawing order', () => {
    const { registry } = setupWithLayout([STATS, TASKS, UPDATER], {
      'titleBar.status': [TASKS_KEY],
      'statusBar.right': [UPDATER_KEY],
    })
    const listKeys = (slot: SlotName) => registry.listLayoutEntries(slot).map((one) => one.key)

    expect(listKeys('titleBar.status')).toStrictEqual([TASKS_KEY, STATS_KEY])
    expect(listKeys('statusBar.right')).toStrictEqual([UPDATER_KEY])
    expect(listKeys('statusBar.left')).toStrictEqual([])
  })

  it('lists the layout entries of a slot with the title of the command, of the view and, failing that, the key', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: LEDGER, slot: 'titleBar.status', command: 'ledger.open' },
        { ...STATS, title: 'Reef statistics' },
        TASKS,
        { pluginId: LEDGER, slot: 'titleBar.status', submenu: 'ledger.more', title: 'More' },
      ],
      { 'titleBar.status': [TASKS_KEY, STATS_KEY] },
    )

    expect(registry.listLayoutEntries('titleBar.status')).toStrictEqual([
      { key: TASKS_KEY, title: TASKS_KEY },
      { key: STATS_KEY, title: 'Reef statistics' },
      { key: `titleBar.status|${LEDGER}/ledger.open`, title: 'Open ledger' },
    ])
    expect(registry.listLayoutEntries('statusBar.left')).toStrictEqual([])
  })

  it('lists the layout keys of a menu slot', () => {
    const { registry } = setupWithLayout(
      [
        { pluginId: LEDGER, slot: 'fileTree.context', command: 'ledger.open' },
        { pluginId: HARBOR, slot: 'fileTree.context', command: 'harbor.open' },
      ],
      undefined,
    )

    expect(registry.listLayoutEntries('fileTree.context').map((one) => one.key)).toStrictEqual([
      `fileTree.context|${HARBOR}/harbor.open`,
      `fileTree.context|${LEDGER}/ledger.open`,
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
