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
  done: 1,
  total: 4,
  counting: '',
  error: '',
  canCancel: true,
  canRetry: false,
  actionLabel: '',
  actionPluginId: '',
}

describe('the slots of a mock context', () => {
  it('hand a watcher the entries a test sets', () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.slots.watchEntries(SLOT.titleBarCommands, handler)

    ctx.slots.setEntries(SLOT.titleBarCommands, [BUTTON])

    expect(handler.mock.calls).toStrictEqual([[[]], [[BUTTON]]])
    expect(ctx.slots.getEntries(SLOT.titleBarCommands)).toStrictEqual([BUTTON])
    expect(ctx.slots).not.toHaveProperty('runCommand')
  })
})

describe('the commands of a mock context', () => {
  const TARGET = { path: 'ledger/March.md', title: 'March', isFolder: false, others: [], tab: '' }

  it('give the items a test sets for a command, and none for another', async () => {
    const ctx = createMockPluginContext()
    const windows = [{ id: 'w2', title: 'Second window' }]

    ctx.commands.setItems('app:tab.move', windows)

    expect(await ctx.commands.listItems('app:tab.move', TARGET)).toStrictEqual(windows)
    expect(await ctx.commands.listItems('app:vault.create')).toStrictEqual([])
  })

  it('record the item id a command runs with', async () => {
    const ctx = createMockPluginContext()

    await ctx.commands.run('app:tab.move', TARGET, 'w2')

    expect(ctx.commands.getRuns()).toStrictEqual([
      { command: 'app:tab.move', target: TARGET, itemId: 'w2' },
    ])
  })

  it('hand a watcher the ids a test sets, at once and on every change', () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.commands.watch(handler)

    ctx.commands.setIds(['app:vault.create', 'org.example.ledger:ledger.open'])

    expect(handler.mock.calls).toStrictEqual([
      [[]],
      [['app:vault.create', 'org.example.ledger:ledger.open']],
    ])
    expect(ctx.commands.getIds()).toStrictEqual([
      'app:vault.create',
      'org.example.ledger:ledger.open',
    ])
  })

  it('record each command run, with its target', async () => {
    const ctx = createMockPluginContext()

    await ctx.commands.run('app:vault.create')
    await ctx.commands.run('org.example.ledger:ledger.open', TARGET)

    expect(ctx.commands.getRuns()).toStrictEqual([
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
    ctx.vaults.watch(vaults)
    ctx.window.watch(window)

    ctx.vaults.setState({
      vaults: [{ id: 'v1', name: 'Harbor' }],
      current: { id: 'v1', name: 'Harbor' },
    })
    ctx.window.setState({ isMaximized: true, isFullscreen: false })
    await ctx.vaults.switchTo('v2')
    ctx.window.minimize()
    ctx.window.toggleMaximize()
    ctx.window.close()

    expect(vaults).toHaveBeenLastCalledWith(ctx.vaults.getState())
    expect(window).toHaveBeenLastCalledWith({ isMaximized: true, isFullscreen: false })
    expect(ctx.vaults.getSwitches()).toStrictEqual(['v2'])
    expect(ctx.window.getActions()).toStrictEqual(['minimize', 'toggleMaximize', 'close'])
  })
})

describe('the background work of a mock context', () => {
  it('passes the tasks a test sets to a watcher, and records each call on them', async () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.backgroundTasks.watch(handler)

    ctx.backgroundTasks.setTasks([SYNC])
    await ctx.backgroundTasks.cancel('sync')
    ctx.backgroundTasks.retry('sync')
    ctx.backgroundTasks.dismiss('sync')
    ctx.backgroundTasks.openSettings('org.example.ledger')

    expect(handler).toHaveBeenLastCalledWith([SYNC])
    expect(ctx.backgroundTasks.getActions()).toStrictEqual([
      { action: 'cancel', id: 'sync' },
      { action: 'retry', id: 'sync' },
      { action: 'dismiss', id: 'sync' },
      { action: 'openSettings', id: 'org.example.ledger' },
    ])
  })
})
