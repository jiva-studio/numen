import { describe, expect, it, vi } from 'vitest'
import type { Vault, VaultsClient } from '../api/vaults'
import { getSyncPercent, useVaults } from './vaults'
import { ReportedError } from '@numen/plugin-sdk'

const NOW = new Date('2026-11-23T12:00:00Z')

const createVault = (patch: Partial<Vault> = {}): Vault => ({
  id: 'Test',
  name: 'Test',
  path: '/vaults/Test',
  state: 'notConnected',
  lastSyncedAt: '',
  conflictCount: 0,
  remoteId: '',
  ...patch,
})

const createClient = (vaults: Vault[] = [createVault()]) => {
  const client = {
    listVaults: vi.fn<VaultsClient['listVaults']>(() => Promise.resolve({ vaults })),
    listRemoteVaults: vi.fn<VaultsClient['listRemoteVaults']>(() =>
      Promise.resolve({
        vaults: [
          { remoteId: 'aa11', name: 'Work', isConnected: false, sizeBytes: 3_670_016 },
          { remoteId: 'bb22', name: 'Home', isConnected: true, sizeBytes: 0 },
        ],
      }),
    ),
    getConnectPreview: vi.fn<VaultsClient['getConnectPreview']>(() =>
      Promise.resolve({ differCount: 0 }),
    ),
    resolveDeletions: vi.fn<VaultsClient['resolveDeletions']>(() => Promise.resolve()),
    connectVault: vi.fn<VaultsClient['connectVault']>(() => Promise.resolve()),
    stopSyncingVault: vi.fn<VaultsClient['stopSyncingVault']>(() => Promise.resolve()),
    deleteCloudVault: vi.fn<VaultsClient['deleteCloudVault']>(() => Promise.resolve()),
    getStatus: vi.fn<VaultsClient['getStatus']>(() =>
      Promise.resolve({ hasIdentity: true, isReachable: true, items: [] }),
    ),
  }
  return client
}

const createModel = (client: ReturnType<typeof createClient>) =>
  useVaults({ client, now: () => NOW })

describe('useVaults', () => {
  it('lists the vaults that are synced, then the vaults of the cloud that this device does not sync', async () => {
    const client = createClient([
      createVault({ id: 'a', name: 'Plain', state: 'notConnected' }),
      createVault({ id: 'b', name: 'Synced', state: 'connected' }),
      createVault({ id: 'c', name: 'Synced copy', state: 'copy' }),
      createVault({ id: 'd', name: 'Received', state: 'received' }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.status.value).toBe('ready')
    expect(model.state.rows.value.map((row) => row.name)).toStrictEqual(['Synced', 'Work'])
  })

  it('says a synced vault is up to date and how long ago it synced', async () => {
    const client = createClient([
      createVault({ state: 'connected', lastSyncedAt: '2026-11-23T11:55:00Z' }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)).toMatchObject({
      status: 'Up to date',
      tone: 'ok',
      kind: 'ok',
      qualifier: 'Last sync 5 minutes ago',
      facts: [],
    })
  })

  it('says how many files stay on demand and the size in the cloud on an idle vault', async () => {
    const client = createClient([
      createVault({
        state: 'connected',
        syncing: false,
        onDemandFiles: 12,
        onDemandBytes: 524_288_000,
        syncedBytes: 1_048_576,
        totalBytes: 525_336_576,
      }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)).toMatchObject({
      status: 'Up to date',
      kind: 'ok',
      facts: ['12 files on demand', '501 MB in the cloud'],
      isTransferring: false,
    })
  })

  it('says one file on demand in the singular', async () => {
    const client = createClient([
      createVault({ state: 'connected', onDemandFiles: 1, totalBytes: 1_048_576 }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)?.facts).toStrictEqual(['1 file on demand', '1 MB in the cloud'])
  })

  it('says nothing about files on demand when none stay in the cloud', async () => {
    const client = createClient([
      createVault({
        state: 'connected',
        onDemandFiles: 0,
        totalBytes: 1_048_576,
        remoteId: 'bb22',
      }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)?.facts.join(' ')).not.toContain('on demand')
  })

  it('leaves the files on demand out of the progress of a syncing vault', async () => {
    const client = createClient([
      createVault({
        state: 'connected',
        syncing: true,
        syncedBytes: 1_000,
        totalBytes: 5_000,
        onDemandBytes: 3_000,
        onDemandFiles: 2,
      }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)).toMatchObject({ progress: 0.5, isTransferring: true })
  })

  it('reaches the full progress when everything meant to download is downloaded', async () => {
    const client = createClient([
      createVault({
        state: 'connected',
        syncing: true,
        syncedBytes: 2_000,
        totalBytes: 5_000,
        onDemandBytes: 3_000,
      }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)?.progress).toBe(1)
  })

  it('counts the percent over the syncing vaults without their files on demand', () => {
    const vaults = [
      createVault({
        id: 'a',
        syncing: true,
        syncedBytes: 100,
        totalBytes: 1_100,
        onDemandBytes: 1_000,
      }),
      createVault({ id: 'b', syncing: true, syncedBytes: 0, totalBytes: 100 }),
    ]

    expect(getSyncPercent(vaults)).toBe(50)
  })

  it('says one minute in the singular and less than a minute for a fresh sync', async () => {
    const client = createClient([
      createVault({ id: 'a', state: 'connected', lastSyncedAt: '2026-11-23T11:59:00Z' }),
      createVault({ id: 'b', state: 'connected', lastSyncedAt: '2026-11-23T11:59:50Z' }),
    ])
    const model = createModel(client)

    await model.actions.load()

    const synced = model.state.rows.value.filter((row) => row.isSynced)
    expect(synced.map((row) => row.qualifier)).toStrictEqual([
      'Last sync 1 minute ago',
      'Last sync less than a minute ago',
    ])
  })

  it('shows no sync time for a vault that has not synced yet', async () => {
    const model = createModel(createClient([createVault({ state: 'connected' })]))

    await model.actions.load()

    expect(model.state.rows.value.at(0)?.qualifier).toBe('')
  })

  it('counts the notes changed on two devices as a problem', async () => {
    const model = createModel(createClient([createVault({ state: 'connected', conflictCount: 2 })]))

    await model.actions.load()

    expect(model.state.rows.value.at(0)).toMatchObject({
      status: 'Needs attention',
      tone: 'alarm',
      kind: 'conflict',
      qualifier: '2 notes changed on two devices',
    })
  })

  it('says the sync waits when the server is unreachable', async () => {
    const client = createClient([createVault({ state: 'connected' })])
    client.getStatus.mockResolvedValue({ hasIdentity: true, isReachable: false, items: [] })
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.at(0)).toMatchObject({
      status: 'Connecting…',
      tone: 'warning',
      kind: 'syncing',
    })
    expect(model.state.canGetFromCloud.value).toBe(false)
  })

  it('shows no vault and the missing identity when the device has no identity', async () => {
    const client = createClient([createVault({ state: 'connected' })])
    client.getStatus.mockResolvedValue({ hasIdentity: false, isReachable: true, items: [] })
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value).toStrictEqual([])
    expect(model.state.warning.value).toBe('No identity on this device')
    expect(client.listVaults).not.toHaveBeenCalled()
  })

  it('keeps the error of a list that cannot be read', async () => {
    const client = createClient()
    client.listVaults.mockRejectedValue(new ReportedError('boom'))
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.status.value).toBe('error')
    expect(model.state.error.value).toBe('boom')
  })

  it('offers the vaults of the identity that this device does not hold', async () => {
    const model = createModel(createClient([createVault({ id: 'a' })]))

    await model.actions.load()

    expect(model.state.cloudVaults.value.map((one) => one.name)).toStrictEqual(['Work'])
    expect(model.state.freeVaults.value.map((one) => one.id)).toStrictEqual(['a'])
    expect(model.state.canGetFromCloud.value).toBe(true)
  })

  it('offers nothing when every vault of the identity is on this device', async () => {
    const client = createClient()
    client.listRemoteVaults.mockResolvedValue({
      vaults: [{ remoteId: 'bb22', name: 'Home', isConnected: true, sizeBytes: 0 }],
    })
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.canGetFromCloud.value).toBe(false)
  })

  it('still lists the vaults when the vaults of the identity cannot be read', async () => {
    const client = createClient([createVault({ state: 'connected' })])
    client.listRemoteVaults.mockRejectedValue(new ReportedError('down'))
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value).toHaveLength(1)
    expect(model.state.cloudVaults.value).toStrictEqual([])
  })

  it('connects a vault to a vault of the identity and reads the vaults again', async () => {
    const client = createClient()
    const model = createModel(client)

    await model.actions.connectExisting('Test', 'aa11')

    expect(client.connectVault).toHaveBeenCalledWith({
      vaultId: 'Test',
      mode: 'existing',
      remoteId: 'aa11',
    })
    expect(client.listVaults).toHaveBeenCalled()
  })

  it('keeps the message of a connection the backend refuses', async () => {
    const client = createClient()
    client.connectVault.mockRejectedValue(new ReportedError('a copy of a connected vault'))
    const model = createModel(client)

    await model.actions.connectExisting('Test', 'aa11')

    expect(model.state.error.value).toBe('a copy of a connected vault')
  })

  describe('connecting to a vault of the identity that holds notes', () => {
    const differ = (count: number) => {
      const client = createClient()
      client.getConnectPreview.mockResolvedValue({ differCount: count })
      return { client, model: createModel(client) }
    }

    it('asks before it connects and keeps the dialog open', async () => {
      const { client, model } = differ(2)

      await model.actions.connectExisting('Test', 'aa11')

      expect(model.state.pendingConfirm.value?.text).toBe(
        '2 notes differ from the vault in the cloud. Get it anyway?',
      )
      expect(client.connectVault).not.toHaveBeenCalled()
    })

    it('says one note in the singular', async () => {
      const { model } = differ(1)

      await model.actions.connectExisting('Test', 'aa11')

      expect(model.state.pendingConfirm.value?.text).toBe(
        '1 note differs from the vault in the cloud. Get it anyway?',
      )
    })

    it('connects once the person confirms', async () => {
      const { client, model } = differ(2)
      await model.actions.connectExisting('Test', 'aa11')

      await model.actions.confirmConnect()

      expect(client.connectVault).toHaveBeenCalledWith({
        vaultId: 'Test',
        mode: 'existing',
        remoteId: 'aa11',
      })
      expect(model.state.pendingConfirm.value).toBeNull()
    })

    it('calls nothing more when the person goes back', async () => {
      const { client, model } = differ(2)
      await model.actions.connectExisting('Test', 'aa11')

      model.actions.cancelConnect()

      expect(model.state.pendingConfirm.value).toBeNull()
      expect(client.connectVault).not.toHaveBeenCalled()
    })

    it('keeps the message of a preview the backend refuses and does not connect', async () => {
      const { client, model } = differ(0)
      client.getConnectPreview.mockRejectedValue(new ReportedError('no preview'))

      await model.actions.connectExisting('Test', 'aa11')

      expect(model.state.error.value).toBe('no preview')
      expect(client.connectVault).not.toHaveBeenCalled()
    })
  })

  it('names the removals that wait for an answer, by the way they go', async () => {
    const client = createClient([
      createVault({
        id: 'a',
        state: 'connected',
        pausedDeletions: { count: 5, tracked: 10, direction: 'send' },
      }),
      createVault({
        id: 'b',
        state: 'connected',
        pausedDeletions: { count: 3, tracked: 9, direction: 'apply' },
      }),
    ])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.map((row) => [row.status, row.canResolve])).toStrictEqual([
      ['Paused', true],
      ['Paused', true],
      ['Not syncing', false],
    ])
  })

  it('sends the answer for the removals and reads the vaults again', async () => {
    const client = createClient()
    const model = createModel(client)

    await model.actions.resolveDeletions('Test', 'keep')

    expect(client.resolveDeletions).toHaveBeenCalledWith({ vaultId: 'Test', choice: 'keep' })
    expect(client.listVaults).toHaveBeenCalled()
  })

  it('asks first, then stops the sync of the vault and lists the vaults again', async () => {
    const client = createClient([createVault({ id: 'a', state: 'connected' })])
    const model = createModel(client)
    await model.actions.load()

    model.actions.askStop(model.state.rows.value[0]!)
    expect(client.stopSyncingVault).not.toHaveBeenCalled()
    await model.actions.confirmStop(false)

    expect(client.stopSyncingVault).toHaveBeenCalledWith({ vaultId: 'a', deleteCloudCopy: false })
    expect(model.state.pendingStop.value).toBeNull()
  })

  it('deletes the cloud copy with the answer, and keeps the vault linked when the server refuses', async () => {
    const client = createClient([createVault({ id: 'a', state: 'connected' })])
    client.stopSyncingVault.mockRejectedValue(new ReportedError('away'))
    const model = createModel(client)
    await model.actions.load()

    model.actions.askStop(model.state.rows.value[0]!)
    await model.actions.confirmStop(true)

    expect(client.stopSyncingVault).toHaveBeenCalledWith({ vaultId: 'a', deleteCloudCopy: true })
    expect(model.state.error.value).toBe('away')
  })

  it('shows the size of a synced vault and of a vault that is not synced here', async () => {
    const client = createClient([createVault({ id: 'a', state: 'connected', remoteId: 'bb22' })])
    const model = createModel(client)

    await model.actions.load()

    expect(model.state.rows.value.map((row) => [row.name, row.facts])).toStrictEqual([
      ['Test', ['0 MB in the cloud']],
      ['Work', ['3.5 MB in the cloud']],
    ])
    expect(model.state.rows.value.map((row) => row.isSynced)).toStrictEqual([true, false])
  })

  it('asks before it deletes the cloud copy of a vault that is not synced here', async () => {
    const client = createClient()
    const model = createModel(client)
    await model.actions.load()

    model.actions.askDelete(model.state.rows.value[0]!)
    expect(client.deleteCloudVault).not.toHaveBeenCalled()
    await model.actions.confirmDelete()

    expect(client.deleteCloudVault).toHaveBeenCalledWith({ remoteId: 'aa11' })
    expect(model.state.pendingDelete.value).toBeNull()
  })

  it('stops nothing when the person cancels', async () => {
    const client = createClient([createVault({ id: 'a', state: 'connected' })])
    const model = createModel(client)
    await model.actions.load()

    model.actions.askStop(model.state.rows.value[0]!)
    model.actions.cancelStop()
    await model.actions.confirmStop(true)

    expect(client.stopSyncingVault).not.toHaveBeenCalled()
  })

  it('keeps the message of an answer the backend refuses', async () => {
    const client = createClient()
    client.resolveDeletions.mockRejectedValue(new ReportedError('refused'))
    const model = createModel(client)

    await model.actions.resolveDeletions('Test', 'remove')

    expect(model.state.error.value).toBe('refused')
  })

  it('shows the generic text for an error that is not a message for display', async () => {
    const client = createClient()
    client.resolveDeletions.mockRejectedValue(new Error('rpc failed at 10.0.0.1'))
    const model = createModel(client)

    await model.actions.resolveDeletions('Test', 'remove')

    expect(model.state.error.value).not.toContain('10.0.0.1')
    expect(model.state.error.value).not.toBe('')
  })
})
