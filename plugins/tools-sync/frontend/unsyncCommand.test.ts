import { afterEach, describe, expect, it, vi } from 'vitest'
import type { MockVaultsHost } from '@numen/plugin-sdk/testing'
import {
  createMockDialogHost,
  createMockHostServices,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate, STOP_SYNC_COMMAND } from './index'
import EndSyncDialog from './ui/EndSyncDialog.vue'

const calls: string[] = []

const run = async (state: 'notConnected' | 'connected' | 'received') => {
  calls.length = 0
  vi.stubGlobal('fetch', (url: string | URL | Request) => {
    const method = String(url).split('/').at(-1) ?? ''
    calls.push(method)
    if (method !== 'ListVaults') return Promise.resolve(Response.json({}))
    const vault = { id: 'v1', name: 'Work', path: '/Work', lastSyncedAt: '', conflictCount: 0 }
    return Promise.resolve(Response.json({ vaults: [{ ...vault, state }] }))
  })
  const dialogs = createMockDialogHost()
  const host = createMockHostServices({ dialogs })
  const current = { id: 'v1', name: 'Work' } as never
  ;(host.vault.vaults as MockVaultsHost).setState({ vaults: [current], current })
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context, host)
  const command = context.getRegisteredCommands().find((one) => one.id === STOP_SYNC_COMMAND)
  await command?.handler({ path: '' } as never)
  return { context, dialogs }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('the command "Stop syncing this vault"', () => {
  it('asks before it stops a synced vault, and stops it on the answer', async () => {
    const { context, dialogs } = await run('connected')

    const opened = dialogs.getOpened().at(0)
    expect(opened?.component).toBe(EndSyncDialog)
    expect(opened?.props).toMatchObject({ kind: 'stop' })
    expect(calls).not.toContain('StopSyncingVault')
    ;(opened?.props?.['onConfirm'] as () => void)()
    await vi.waitFor(() => expect(calls).toContain('StopSyncingVault'))
    expect(dialogs.getOpened()).toHaveLength(0)
    context.dispose()
  })

  it('stops nothing when the question is cancelled', async () => {
    const { context, dialogs } = await run('connected')

    ;(dialogs.getOpened().at(0)?.props?.['onCancel'] as () => void)()

    expect(calls).not.toContain('StopSyncingVault')
    expect(dialogs.getOpened()).toHaveLength(0)
    context.dispose()
  })

  it('asks to leave when the vault is a received folder', async () => {
    const { context, dialogs } = await run('received')

    expect(dialogs.getOpened().at(0)?.props).toMatchObject({ kind: 'leave' })
    context.dispose()
  })

  it('opens no question for a vault that is not synced', async () => {
    const { context, dialogs } = await run('notConnected')

    expect(dialogs.getOpened()).toHaveLength(0)
    context.dispose()
  })
})
