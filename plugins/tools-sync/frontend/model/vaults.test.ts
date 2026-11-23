import { describe, expect, it, vi } from 'vitest'
import type { ListVaultsResponse, Vault, VaultsClient } from '../api/vaults'
import { useVaults } from './vaults'

const NOW = new Date('2026-11-23T12:00:00Z')

const createVault = (patch: Partial<Vault> = {}): Vault => ({
  id: 'Test',
  name: 'Test',
  path: '/vaults/Test',
  state: 'notConnected',
  lastSyncedAt: '',
  conflictCount: 0,
  ...patch,
})

const createClient = (vaults: Vault[] = [createVault()]) => {
  const client = {
    listVaults: vi.fn<VaultsClient['listVaults']>(() => Promise.resolve({ vaults })),
    listRemoteVaults: vi.fn<VaultsClient['listRemoteVaults']>(() =>
      Promise.resolve({
        vaults: [
          { remoteId: 'aa11', name: 'Work', isConnected: false },
          { remoteId: 'bb22', name: 'Home', isConnected: true },
        ],
      }),
    ),
    connectVault: vi.fn<VaultsClient['connectVault']>(() => Promise.resolve()),
    getStatus: vi.fn<VaultsClient['getStatus']>(() =>
      Promise.resolve({ hasIdentity: true, isReachable: true, items: [] }),
    ),
  }
  return client
}

const createModel = (client: ReturnType<typeof createClient>) =>
  useVaults({ client, now: () => NOW })

describe('useVaults', () => {
  it('loads the vaults and names the state of each', async () => {
    const client = createClient([
      createVault({ id: 'a', name: 'Plain', state: 'notConnected' }),
      createVault({ id: 'b', name: 'Synced', state: 'connected' }),
      createVault({ id: 'c', name: 'Synced copy', state: 'copy' }),
    ])
    const model = createModel(client)

    await model.load()

    expect(model.status.value).toBe('ready')
    expect(model.rows.value.map((row) => row.stateLabel)).toStrictEqual([
      'Not connected',
      'Connected',
      'a copy of a connected vault',
    ])
    expect(model.rows.value.map((row) => row.name)).toStrictEqual(['Plain', 'Synced', 'Synced copy'])
  })

  it('says how long ago a connected vault synced', async () => {
    const client = createClient([
      createVault({ state: 'connected', lastSyncedAt: '2026-11-23T11:55:00Z' }),
    ])
    const model = createModel(client)

    await model.load()

    expect(model.rows.value.at(0)?.syncedLabel).toBe('Synced 5 minutes ago')
  })

  it('says one minute in the singular', async () => {
    const client = createClient([
      createVault({ state: 'connected', lastSyncedAt: '2026-11-23T11:59:00Z' }),
    ])
    const model = createModel(client)

    await model.load()

    expect(model.rows.value.at(0)?.syncedLabel).toBe('Synced 1 minute ago')
  })

  it('shows no sync time for a vault that has not synced', async () => {
    const client = createClient([createVault({ state: 'connected', lastSyncedAt: '' })])
    const model = createModel(client)

    await model.load()

    expect(model.rows.value.at(0)?.syncedLabel).toBe('')
  })

  it('counts the notes changed on two computers', async () => {
    const client = createClient([
      createVault({ id: 'a', state: 'connected', conflictCount: 1 }),
      createVault({ id: 'b', state: 'connected', conflictCount: 2 }),
      createVault({ id: 'c', state: 'connected', conflictCount: 0 }),
    ])
    const model = createModel(client)

    await model.load()

    expect(model.rows.value.map((row) => row.conflictLabel)).toStrictEqual([
      '1 note changed on two computers',
      '2 notes changed on two computers',
      '',
    ])
  })

  it('offers to connect a vault that is not connected, and no other', async () => {
    const client = createClient([
      createVault({ id: 'a', state: 'notConnected' }),
      createVault({ id: 'b', state: 'connected' }),
      createVault({ id: 'c', state: 'copy' }),
    ])
    const model = createModel(client)

    await model.load()

    expect(model.rows.value.map((row) => row.canConnect)).toStrictEqual([true, false, false])
  })

  it('shows no vault and the missing identity when the device has no identity', async () => {
    const client = createClient()
    client.getStatus.mockResolvedValue({
      hasIdentity: false,
      isReachable: true,
      items: [{ key: 'no-identity', text: 'No identity on this device' }],
    })
    const model = createModel(client)

    await model.load()

    expect(model.hasIdentity.value).toBe(false)
    expect(model.warning.value).toBe('No identity on this device')
    expect(model.rows.value).toStrictEqual([])
  })

  it('keeps the error of a list that cannot be read', async () => {
    const client = createClient()
    client.listVaults.mockRejectedValue(new Error('the sync backend is not running'))
    const model = createModel(client)

    await model.load()

    expect(model.status.value).toBe('error')
    expect(model.error.value).toBe('the sync backend is not running')
  })

  it('opens the list of synced vaults of the account for one vault', async () => {
    const client = createClient()
    const model = createModel(client)
    await model.load()

    await model.openRemote('Test')

    expect(model.openFor.value).toBe('Test')
    expect(model.remote.value.map((one) => one.name)).toStrictEqual(['Work', 'Home'])
  })

  it('shows an empty list of synced vaults', async () => {
    const client = createClient()
    client.listRemoteVaults.mockResolvedValue({ vaults: [] })
    const model = createModel(client)
    await model.load()

    await model.openRemote('Test')

    expect(model.openFor.value).toBe('Test')
    expect(model.remote.value).toStrictEqual([])
  })

  it('closes the list of synced vaults', async () => {
    const client = createClient()
    const model = createModel(client)
    await model.load()
    await model.openRemote('Test')

    model.closeRemote()

    expect(model.openFor.value).toBe('')
  })

  it('connects a vault as a new synced vault and reads the vaults again', async () => {
    const client = createClient()
    const model = createModel(client)
    await model.load()
    client.listVaults.mockResolvedValue({
      vaults: [createVault({ state: 'connected' })],
    } satisfies ListVaultsResponse)

    await model.connectNew('Test')

    expect(client.connectVault).toHaveBeenCalledWith({ vaultId: 'Test', mode: 'new' })
    expect(model.rows.value.at(0)?.stateLabel).toBe('Connected')
  })

  it('connects a vault to a synced vault, closes the list and reads the vaults again', async () => {
    const client = createClient()
    const model = createModel(client)
    await model.load()
    await model.openRemote('Test')
    client.listVaults.mockResolvedValue({ vaults: [createVault({ state: 'connected' })] })

    await model.connectExisting('Test', 'aa11')

    expect(client.connectVault).toHaveBeenCalledWith({
      vaultId: 'Test',
      mode: 'existing',
      remoteId: 'aa11',
    })
    expect(model.openFor.value).toBe('')
    expect(model.rows.value.at(0)?.stateLabel).toBe('Connected')
  })

  it('keeps the message of a connection the backend refuses', async () => {
    const client = createClient()
    client.connectVault.mockRejectedValue(new Error('a copy of a connected vault'))
    const model = createModel(client)
    await model.load()

    await model.connectNew('Test')

    expect(model.error.value).toBe('a copy of a connected vault')
    expect(model.status.value).toBe('ready')
  })

  it('clears the message of a refusal when the next connection works', async () => {
    const client = createClient()
    client.connectVault.mockRejectedValueOnce(new Error('a copy of a connected vault'))
    const model = createModel(client)
    await model.load()
    await model.connectNew('Test')

    await model.connectNew('Test')

    expect(model.error.value).toBe('')
  })
})
