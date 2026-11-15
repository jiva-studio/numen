/** The stand-ins of the slots, the vaults, the window and the background work a plugin test drives. */
import { describe, expect, it, vi } from 'vitest'
import { createMockHostServices, SLOT, type BackgroundTask, type SlotItem } from '../src'

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
    const host = createMockHostServices()
    const handler = vi.fn()
    host.ui.slots.watchEntries(SLOT.titleBarCommands, handler)

    host.ui.slots.setEntries(SLOT.titleBarCommands, [BUTTON])

    expect(handler.mock.calls).toStrictEqual([[[]], [[BUTTON]]])
    expect(host.ui.slots.getEntries(SLOT.titleBarCommands)).toStrictEqual([BUTTON])
    expect(host.ui.slots).not.toHaveProperty('runCommand')
  })
})

describe('the commands of a mock context', () => {
  const TARGET = { path: 'ledger/March.md', title: 'March', isFolder: false, others: [], tab: '' }

  it('give the items a test sets for a command, and none for another', async () => {
    const host = createMockHostServices()
    const windows = [{ id: 'w2', title: 'Second window' }]

    host.ui.commands.setItems('app:tab.move', windows)

    expect(await host.ui.commands.listItems('app:tab.move', TARGET)).toStrictEqual(windows)
    expect(await host.ui.commands.listItems('app:vault.create')).toStrictEqual([])
  })

  it('record the item id a command runs with', async () => {
    const host = createMockHostServices()

    await host.ui.commands.run('app:tab.move', TARGET, 'w2')

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: 'app:tab.move', target: TARGET, itemId: 'w2' },
    ])
  })

  it('resolve a run with the path a test sets for the command, and an empty path for any other', async () => {
    const host = createMockHostServices()
    host.ui.commands.setResult('app:note.create', 'Untitled note 1.md')

    expect(await host.ui.commands.run('app:note.create', TARGET)).toBe('Untitled note 1.md')
    expect(await host.ui.commands.run('app:note.open', TARGET)).toBe('')
  })

  it('hand a watcher the ids a test sets, at once and on every change', () => {
    const host = createMockHostServices()
    const handler = vi.fn()
    host.ui.commands.watch(handler)

    host.ui.commands.setIds(['app:vault.create', 'org.example.ledger:ledger.open'])

    expect(handler.mock.calls).toStrictEqual([
      [[]],
      [['app:vault.create', 'org.example.ledger:ledger.open']],
    ])
    expect(host.ui.commands.getIds()).toStrictEqual([
      'app:vault.create',
      'org.example.ledger:ledger.open',
    ])
  })

  it('record each command run, with its target', async () => {
    const host = createMockHostServices()

    await host.ui.commands.run('app:vault.create')
    await host.ui.commands.run('org.example.ledger:ledger.open', TARGET)

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: 'app:vault.create', target: undefined, itemId: undefined },
      { command: 'org.example.ledger:ledger.open', target: TARGET, itemId: undefined },
    ])
  })
})

describe('the vaults and the window of a mock context', () => {
  it('hand a watcher the state a test sets, and record switches and button presses', async () => {
    const host = createMockHostServices()
    const vaults = vi.fn()
    const window = vi.fn()
    host.vault.vaults.watch(vaults)
    host.ui.window.watch(window)

    host.vault.vaults.setState({
      vaults: [{ id: 'v1', name: 'Harbor' }],
      current: { id: 'v1', name: 'Harbor' },
    })
    host.ui.window.setState({ isMaximized: true, isFullscreen: false })
    await host.vault.vaults.switchTo('v2')
    host.ui.window.minimize()
    host.ui.window.toggleMaximize()
    host.ui.window.close()

    expect(vaults).toHaveBeenLastCalledWith(host.vault.vaults.getState())
    expect(window).toHaveBeenLastCalledWith({ isMaximized: true, isFullscreen: false })
    expect(host.vault.vaults.getSwitches()).toStrictEqual(['v2'])
    expect(host.ui.window.getActions()).toStrictEqual(['minimize', 'toggleMaximize', 'close'])
  })
})

describe('the background work of a mock context', () => {
  it('passes the tasks a test sets to a watcher, and records each call on them', async () => {
    const host = createMockHostServices()
    const handler = vi.fn()
    host.work.backgroundTasks.watch(handler)

    host.work.backgroundTasks.setTasks([SYNC])
    await host.work.backgroundTasks.cancel('sync')
    host.work.backgroundTasks.retry('sync')
    host.work.backgroundTasks.dismiss('sync')
    host.work.backgroundTasks.runAction('sync')

    expect(handler).toHaveBeenLastCalledWith([SYNC])
    expect(host.work.backgroundTasks.getActions()).toStrictEqual([
      { action: 'cancel', id: 'sync' },
      { action: 'retry', id: 'sync' },
      { action: 'dismiss', id: 'sync' },
      { action: 'runAction', id: 'sync' },
    ])
  })
})
