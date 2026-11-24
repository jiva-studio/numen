import { describe, expect, it, vi } from 'vitest'
import type { VaultsClient } from '../api/vaults'
import { createSyncedFlags } from './syncedFlags'

type Listed = Awaited<ReturnType<VaultsClient['listVaults']>>['vaults'][number]

const vault = (id: string, state: Listed['state']): Listed => ({
  id,
  name: id,
  path: `/${id}`,
  state,
  lastSyncedAt: '',
  remoteId: `remote-${id}`,
  conflictCount: 0,
})

const setup = (listVaults: VaultsClient['listVaults']) => {
  const set = vi.fn<(vaultId: string, key: string, value: boolean) => void>()
  const synced = createSyncedFlags({
    client: { listVaults },
    flags: { setFlag: set },
    getVaultIds: () => ['sea', 'hill', 'lake'],
  })
  return { set, synced }
}

describe('the isSynced flag of the vaults', () => {
  it('is true for a connected vault and false for any other vault of the window', async () => {
    const { set, synced } = setup(() =>
      Promise.resolve({ vaults: [vault('sea', 'connected'), vault('hill', 'notConnected')] }),
    )

    await synced.refresh()

    expect(set.mock.calls).toStrictEqual([
      ['sea', 'isSynced', true],
      ['hill', 'isSynced', false],
      ['lake', 'isSynced', false],
    ])
  })

  it('is false for every vault where the list cannot be read', async () => {
    const { set, synced } = setup(() => Promise.reject(new Error('no identity')))

    await synced.refresh()

    expect(set.mock.calls.map((one) => one[2])).toStrictEqual([false, false, false])
  })
})
