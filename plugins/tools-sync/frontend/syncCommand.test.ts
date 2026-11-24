import { afterEach, describe, expect, it, vi } from 'vitest'
import type { MockVaultsHost } from '@numen/plugin-sdk/testing'
import {
  createMockDialogHost,
  createMockHostServices,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate, SYNC_COMMAND } from './index'
import SyncChoiceDialog from './ui/SyncChoiceDialog.vue'

interface Server {
  readonly cloud: { remoteId: string; name: string; isConnected: boolean }[]
  readonly state: 'notConnected' | 'connected'
}

const calls: string[] = []

const stubServer = (server: Server) => {
  vi.stubGlobal('fetch', (url: string | URL | Request, init?: RequestInit) => {
    const method = String(url).split('/').at(-1) ?? ''
    calls.push(method === 'ConnectVault' ? `${method} ${String(init?.body)}` : method)
    switch (method) {
      case 'ListVaults':
        return Promise.resolve(
          Response.json({
            vaults: [
              {
                id: 'v1',
                name: 'Work',
                path: '/Work',
                state: server.state,
                lastSyncedAt: '',
                conflictCount: 0,
              },
            ],
          }),
        )
      case 'ListRemoteVaults':
        return Promise.resolve(Response.json({ vaults: server.cloud }))
      default:
        return Promise.resolve(Response.json({}))
    }
  })
}

const run = async (server: Server) => {
  calls.length = 0
  stubServer(server)
  const dialogs = createMockDialogHost()
  const host = createMockHostServices({ dialogs })
  const current = { id: 'v1', name: 'Work' }
  ;(host.vault.vaults as MockVaultsHost).setState({ vaults: [current], current })
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context, host)
  const command = context.getRegisteredCommands().find((one) => one.id === SYNC_COMMAND)
  await command?.handler({ path: '' } as never)
  return { context, dialogs }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('the command "Sync this vault"', () => {
  it('sends the vault as a new vault when the identity holds none of that name', async () => {
    const { context, dialogs } = await run({ cloud: [], state: 'notConnected' })

    expect(calls.some((one) => one.startsWith('ConnectVault'))).toBe(true)
    expect(dialogs.getOpened()).toHaveLength(0)
    context.dispose()
  })

  it('connects nothing for a vault that is synced already', async () => {
    const { context } = await run({ cloud: [], state: 'connected' })

    expect(calls.some((one) => one.startsWith('ConnectVault'))).toBe(false)
    context.dispose()
  })

  it('asks which copy to use when the identity holds a vault of the same name', async () => {
    const { context, dialogs } = await run({
      cloud: [{ remoteId: 'aa11', name: 'Work', isConnected: false }],
      state: 'notConnected',
    })

    expect(dialogs.getOpened().at(0)?.component).toBe(SyncChoiceDialog)
    expect(calls.some((one) => one.startsWith('ConnectVault'))).toBe(false)
    context.dispose()
  })
})
