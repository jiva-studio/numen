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
  it('hand a watcher the entries a test sets, and record the commands run', async () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.slots.watchEntries(SLOT.titleBarCommands, handler)

    ctx.slots.setEntries(SLOT.titleBarCommands, [BUTTON])
    await ctx.slots.runCommand(BUTTON.kind === 'command' ? BUTTON.command : '')

    expect(handler.mock.calls).toStrictEqual([[[]], [[BUTTON]]])
    expect(ctx.slots.getEntries(SLOT.titleBarCommands)).toStrictEqual([BUTTON])
    expect(ctx.slots.getRuns()).toStrictEqual([
      { command: 'org.example.ledger:ledger.open', target: undefined },
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
