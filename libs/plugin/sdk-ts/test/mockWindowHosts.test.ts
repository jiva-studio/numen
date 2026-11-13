/** The stand-ins of the slots, the vaults, the window and the background work a plugin test drives. */
import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext, SLOT, type BackgroundTask, type SlotItem } from '../src'

const BUTTON: SlotItem = {
  kind: 'command',
  key: 'org.example.ledger/command/ledger.open',
  pluginId: 'org.example.ledger',
  command: 'org.example.ledger:ledger.open',
  title: 'Open ledger',
  shortcut: '',
  hasItems: false,
}

const SYNC: BackgroundTask = {
  id: 'sync',
  label: 'Syncing the ledger',
  about: '',
  state: 'running',
  progress: { completedCount: 1, total: 4, counting: '' },
  error: '',
  allowed: { cancel: true, retry: false },
  actionLabel: '',
}

describe('the slots of a mock context', () => {
  it('hand a watcher the entries a test sets', () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.ui.slots.watchEntries(SLOT.titleBarCommands, handler)

    ctx.ui.slots.setEntries(SLOT.titleBarCommands, [BUTTON])

    expect(handler.mock.calls).toStrictEqual([[[]], [[BUTTON]]])
    expect(ctx.ui.slots.getEntries(SLOT.titleBarCommands)).toStrictEqual([BUTTON])
    expect(ctx.ui.slots).not.toHaveProperty('runCommand')
  })
})

describe('the commands of a mock context', () => {
  const TARGET = { path: 'ledger/March.md', title: 'March', isFolder: false, others: [], tab: '' }

  it('give the items a test sets for a command, and none for another', async () => {
    const ctx = createMockPluginContext()
    const windows = [{ id: 'w2', title: 'Second window' }]

    ctx.ui.commands.setItems('app:tab.move', windows)

    expect(await ctx.ui.commands.listItems('app:tab.move', TARGET)).toStrictEqual(windows)
    expect(await ctx.ui.commands.listItems('app:vault.create')).toStrictEqual([])
  })

  it('record the item id a command runs with', async () => {
    const ctx = createMockPluginContext()

    await ctx.ui.commands.run('app:tab.move', TARGET, 'w2')

    expect(ctx.ui.commands.getRuns()).toStrictEqual([
      { command: 'app:tab.move', target: TARGET, itemId: 'w2' },
    ])
  })

  it('resolve a run with the path a test sets for the command, and an empty path for any other', async () => {
    const ctx = createMockPluginContext()
    ctx.ui.commands.setResult('app:note.create', 'Untitled note 1.md')

    expect(await ctx.ui.commands.run('app:note.create', TARGET)).toBe('Untitled note 1.md')
    expect(await ctx.ui.commands.run('app:note.open', TARGET)).toBe('')
  })

  it('hand a watcher the ids a test sets, at once and on every change', () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.ui.commands.watch(handler)

    ctx.ui.commands.setIds(['app:vault.create', 'org.example.ledger:ledger.open'])

    expect(handler.mock.calls).toStrictEqual([
      [[]],
      [['app:vault.create', 'org.example.ledger:ledger.open']],
    ])
    expect(ctx.ui.commands.getIds()).toStrictEqual([
      'app:vault.create',
      'org.example.ledger:ledger.open',
    ])
  })

  it('record each command run, with its target', async () => {
    const ctx = createMockPluginContext()

    await ctx.ui.commands.run('app:vault.create')
    await ctx.ui.commands.run('org.example.ledger:ledger.open', TARGET)

    expect(ctx.ui.commands.getRuns()).toStrictEqual([
      { command: 'app:vault.create', target: undefined, itemId: undefined },
      { command: 'org.example.ledger:ledger.open', target: TARGET, itemId: undefined },
    ])
  })
})

describe('the vaults and the window of a mock context', () => {
  it('hand a watcher the state a test sets, and record switches and button presses', async () => {
    const ctx = createMockPluginContext()
    const vaults = vi.fn()
    const window = vi.fn()
    ctx.vault.vaults.watch(vaults)
    ctx.ui.window.watch(window)

    ctx.vault.vaults.setState({
      vaults: [{ id: 'v1', name: 'Harbor' }],
      current: { id: 'v1', name: 'Harbor' },
    })
    ctx.ui.window.setState({ isMaximized: true, isFullscreen: false })
    await ctx.vault.vaults.switchTo('v2')
    ctx.ui.window.minimize()
    ctx.ui.window.toggleMaximize()
    ctx.ui.window.close()

    expect(vaults).toHaveBeenLastCalledWith(ctx.vault.vaults.getState())
    expect(window).toHaveBeenLastCalledWith({ isMaximized: true, isFullscreen: false })
    expect(ctx.vault.vaults.getSwitches()).toStrictEqual(['v2'])
    expect(ctx.ui.window.getActions()).toStrictEqual(['minimize', 'toggleMaximize', 'close'])
  })
})

describe('the background work of a mock context', () => {
  it('passes the tasks a test sets to a watcher, and records each call on them', async () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.work.backgroundTasks.watch(handler)

    ctx.work.backgroundTasks.setTasks([SYNC])
    await ctx.work.backgroundTasks.cancel('sync')
    ctx.work.backgroundTasks.retry('sync')
    ctx.work.backgroundTasks.dismiss('sync')
    ctx.work.backgroundTasks.runAction('sync')

    expect(handler).toHaveBeenLastCalledWith([SYNC])
    expect(ctx.work.backgroundTasks.getActions()).toStrictEqual([
      { action: 'cancel', id: 'sync' },
      { action: 'retry', id: 'sync' },
      { action: 'dismiss', id: 'sync' },
      { action: 'runAction', id: 'sync' },
    ])
  })
})
