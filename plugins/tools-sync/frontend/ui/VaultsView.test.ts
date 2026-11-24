import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { RemoteVault, Vault, VaultsClient } from '../api/vaults'
import VaultsView from './VaultsView.vue'

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

interface Setup {
  readonly vaults?: Vault[]
  readonly remote?: RemoteVault[]
  readonly hasIdentity?: boolean
  readonly differCount?: number
}

const createClient = (setup: Setup = {}) => ({
  listVaults: vi.fn<VaultsClient['listVaults']>(() =>
    Promise.resolve({ vaults: setup.vaults ?? [createVault()] }),
  ),
  listRemoteVaults: vi.fn<VaultsClient['listRemoteVaults']>(() =>
    Promise.resolve({
      vaults: setup.remote ?? [{ remoteId: 'aa11', name: 'Work', isConnected: false }],
    }),
  ),
  getConnectPreview: vi.fn<VaultsClient['getConnectPreview']>(() =>
    Promise.resolve({ differCount: setup.differCount ?? 0 }),
  ),
  resolveDeletions: vi.fn<VaultsClient['resolveDeletions']>(() => Promise.resolve()),
  connectVault: vi.fn<VaultsClient['connectVault']>(() => Promise.resolve()),
  getStatus: vi.fn<VaultsClient['getStatus']>(() =>
    Promise.resolve({ hasIdentity: setup.hasIdentity ?? true, isReachable: true, items: [] }),
  ),
})

const render = async (setup: Setup = {}) => {
  const client = createClient(setup)
  const wrapper = mount(VaultsView, { props: { client, now: () => NOW } })
  await flushPromises()
  return { wrapper, client }
}

const findButton = (wrapper: VueWrapper, text: string) =>
  wrapper.findAll('button').find((one) => one.text() === text)

describe('VaultsView', () => {
  it('reads the vaults again while it stays open and stops when it closes', async () => {
    vi.useFakeTimers()
    try {
      const { wrapper, client } = await render({ hasIdentity: false })
      expect(wrapper.text()).toContain('No identity on this device')
      client.getStatus.mockResolvedValue({ hasIdentity: true, isReachable: true, items: [] })

      await vi.advanceTimersByTimeAsync(15_000)
      await flushPromises()

      expect(wrapper.text()).not.toContain('No identity on this device')
      wrapper.unmount()
      const calls = client.getStatus.mock.calls.length
      await vi.advanceTimersByTimeAsync(60_000)
      expect(client.getStatus.mock.calls.length).toBe(calls)
    } finally {
      vi.useRealTimers()
    }
  })

  it('says what to do when no vault is synced and lists no vault of the device', async () => {
    const { wrapper } = await render({ vaults: [createVault({ name: 'Plain' })], remote: [] })

    expect(wrapper.text()).toContain('No vault is synced yet')
    expect(wrapper.text()).toContain(
      'Right-click the top folder of a vault in the file tree and choose Sync this vault.',
    )
    expect(wrapper.text()).not.toContain('Plain')
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('lists the synced vaults with their status', async () => {
    const { wrapper } = await render({
      vaults: [
        createVault({ id: 'a', name: 'Plain' }),
        createVault({ id: 'b', name: 'Synced', state: 'connected' }),
      ],
      remote: [],
    })

    expect(wrapper.text()).toContain('Synced')
    expect(wrapper.text()).toContain('Up to date')
    expect(wrapper.text()).not.toContain('Plain')
    expect(wrapper.text()).not.toContain('Not connected')
  })

  it('offers to get a vault from the cloud only when the account holds one this device lacks', async () => {
    const withCloud = await render({ vaults: [createVault()] })
    const withoutCloud = await render({ vaults: [createVault()], remote: [] })

    expect(findButton(withCloud.wrapper, 'Get a vault from the cloud')).toBeDefined()
    expect(findButton(withoutCloud.wrapper, 'Get a vault from the cloud')).toBeUndefined()
  })

  it('gets the chosen vault of the cloud into the chosen vault of the device', async () => {
    const { wrapper, client } = await render({ vaults: [createVault({ id: 'a' })] })

    await findButton(wrapper, 'Get a vault from the cloud')?.trigger('click')
    await findButton(wrapper, 'Get vault')?.trigger('click')
    await flushPromises()

    expect(client.connectVault).toHaveBeenCalledWith({
      vaultId: 'a',
      mode: 'existing',
      remoteId: 'aa11',
    })
  })

  it('asks before it gets a vault whose notes differ and goes back on a click', async () => {
    const { wrapper, client } = await render({ vaults: [createVault()], differCount: 2 })

    await findButton(wrapper, 'Get a vault from the cloud')?.trigger('click')
    await findButton(wrapper, 'Get vault')?.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('2 notes differ from the vault in the cloud. Get it anyway?')
    expect(client.connectVault).not.toHaveBeenCalled()

    await findButton(wrapper, 'Cancel')?.trigger('click')

    expect(wrapper.text()).toContain('Vault in the cloud')
  })

  it('offers the removals that wait for an answer', async () => {
    const { wrapper, client } = await render({
      vaults: [
        createVault({
          state: 'connected',
          pausedDeletions: { count: 5, tracked: 10, direction: 'send' },
        }),
      ],
      remote: [],
    })

    expect(wrapper.text()).toContain('Paused: 5 of 10 files are removed here')
    await findButton(wrapper, 'Keep the files')?.trigger('click')

    expect(client.resolveDeletions).toHaveBeenCalledWith({ vaultId: 'Test', choice: 'keep' })
  })

  it('offers to try again when the vaults cannot be read', async () => {
    const client = createClient()
    client.listVaults.mockRejectedValueOnce(new Error('boom'))
    const wrapper = mount(VaultsView, { props: { client, now: () => NOW } })
    await flushPromises()
    expect(wrapper.text()).toContain('boom')

    await findButton(wrapper, 'Try again')?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('boom')
  })
})
