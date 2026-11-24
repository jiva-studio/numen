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

  it('lists every vault with its path and state', async () => {
    const { wrapper } = await render({
      vaults: [
        createVault({ id: 'a', name: 'Plain', path: '/vaults/Plain' }),
        createVault({ id: 'b', name: 'Synced', path: '/vaults/Synced', state: 'connected' }),
      ],
    })

    const text = wrapper.text()

    expect(text).toContain('Vaults')
    expect(text).toContain('Plain')
    expect(text).toContain('/vaults/Plain')
    expect(text).toContain('Not connected')
    expect(text).toContain('Synced')
    expect(text).toContain('Connected')
  })

  it('shows when a connected vault synced last', async () => {
    const { wrapper } = await render({
      vaults: [createVault({ state: 'connected', lastSyncedAt: '2026-11-23T11:55:00Z' })],
    })

    expect(wrapper.text()).toContain('Synced 5 minutes ago')
  })

  it('shows how many notes changed on two computers', async () => {
    const { wrapper } = await render({
      vaults: [createVault({ state: 'connected', conflictCount: 2 })],
    })

    expect(wrapper.text()).toContain('2 notes changed on two computers')
  })

  it('offers both ways to connect a vault that is not connected', async () => {
    const { wrapper } = await render()

    expect(findButton(wrapper, 'Connect as a new synced vault')).toBeDefined()
    expect(findButton(wrapper, 'Connect to an existing synced vault')).toBeDefined()
  })

  it('offers no connection for a connected vault', async () => {
    const { wrapper } = await render({ vaults: [createVault({ state: 'connected' })] })

    expect(findButton(wrapper, 'Connect as a new synced vault')).toBeUndefined()
    expect(findButton(wrapper, 'Connect to an existing synced vault')).toBeUndefined()
  })

  it('connects the vault as a new synced vault', async () => {
    const { wrapper, client } = await render()

    await findButton(wrapper, 'Connect as a new synced vault')?.trigger('click')
    await flushPromises()

    expect(client.connectVault).toHaveBeenCalledWith({ vaultId: 'Test', mode: 'new' })
  })

  it('lists the synced vaults of the account to choose from', async () => {
    const { wrapper } = await render({
      remote: [
        { remoteId: 'aa11', name: 'Work', isConnected: false },
        { remoteId: 'bb22', name: 'Home', isConnected: false },
      ],
    })

    await findButton(wrapper, 'Connect to an existing synced vault')?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Choose a synced vault')
    expect(findButton(wrapper, 'Work')).toBeDefined()
    expect(findButton(wrapper, 'Home')).toBeDefined()
  })

  it('connects the vault to the synced vault that is chosen', async () => {
    const { wrapper, client } = await render()
    await findButton(wrapper, 'Connect to an existing synced vault')?.trigger('click')
    await flushPromises()

    await findButton(wrapper, 'Work')?.trigger('click')
    await flushPromises()

    expect(client.connectVault).toHaveBeenCalledWith({
      vaultId: 'Test',
      mode: 'existing',
      remoteId: 'aa11',
    })
    expect(wrapper.text()).not.toContain('Choose a synced vault')
  })

  it('says so when the account has no synced vault', async () => {
    const { wrapper } = await render({ remote: [] })

    await findButton(wrapper, 'Connect to an existing synced vault')?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('No synced vaults in your account')
  })

  it('warns that a vault is a copy of a connected vault', async () => {
    const { wrapper } = await render({ vaults: [createVault({ state: 'copy' })] })

    expect(wrapper.text()).toContain('a copy of a connected vault')
    expect(findButton(wrapper, 'Connect as a new synced vault')).toBeUndefined()
  })

  it('warns when the backend refuses to connect a copy', async () => {
    const { wrapper, client } = await render()
    client.connectVault.mockRejectedValue(new Error('a copy of a connected vault'))

    await findButton(wrapper, 'Connect as a new synced vault')?.trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('a copy of a connected vault')
  })

  it('warns about the missing identity and offers no connection', async () => {
    const { wrapper } = await render({ hasIdentity: false })

    expect(wrapper.text()).toContain('No identity on this device')
    expect(findButton(wrapper, 'Connect as a new synced vault')).toBeUndefined()
    expect(findButton(wrapper, 'Connect to an existing synced vault')).toBeUndefined()
  })

  describe('a synced vault with notes that differ', () => {
    const choose = async (setup: Setup) => {
      const rendered = await render(setup)
      await findButton(rendered.wrapper, 'Connect to an existing synced vault')?.trigger('click')
      await flushPromises()
      await findButton(rendered.wrapper, 'Work')?.trigger('click')
      await flushPromises()
      return rendered
    }

    it('asks with the number of notes that differ before it connects', async () => {
      const { wrapper, client } = await choose({ differCount: 2 })

      expect(wrapper.text()).toContain('2 notes differ from the synced vault')
      expect(findButton(wrapper, 'Connect')).toBeDefined()
      expect(findButton(wrapper, 'Cancel')).toBeDefined()
      expect(client.connectVault).not.toHaveBeenCalled()
    })

    it('connects when the person answers Connect', async () => {
      const { wrapper, client } = await choose({ differCount: 2 })

      await findButton(wrapper, 'Connect')?.trigger('click')
      await flushPromises()

      expect(client.connectVault).toHaveBeenCalledWith({
        vaultId: 'Test',
        mode: 'existing',
        remoteId: 'aa11',
      })
      expect(wrapper.text()).not.toContain('differ from the synced vault')
    })

    it('connects nothing when the person answers Cancel', async () => {
      const { wrapper, client } = await choose({ differCount: 1 })

      await findButton(wrapper, 'Cancel')?.trigger('click')
      await flushPromises()

      expect(client.connectVault).not.toHaveBeenCalled()
      expect(wrapper.text()).not.toContain('differs from the synced vault')
      expect(wrapper.text()).toContain('Not connected')
    })

    it('asks nothing when no note differs', async () => {
      const { wrapper, client } = await choose({ differCount: 0 })

      expect(wrapper.text()).not.toContain('from the synced vault')
      expect(client.connectVault).toHaveBeenCalledTimes(1)
    })
  })

  it('shows the paused removals of a vault and sends the answer chosen', async () => {
    const { wrapper, client } = await render({
      vaults: [
        createVault({
          state: 'connected',
          pausedDeletions: { count: 5, tracked: 10, direction: 'send' },
        }),
      ],
    })

    expect(wrapper.text()).toContain('Sync is paused: 5 of 10 files are removed')
    await findButton(wrapper, 'Keep the files')?.trigger('click')
    await flushPromises()

    expect(client.resolveDeletions).toHaveBeenCalledWith({ vaultId: 'Test', choice: 'keep' })
  })

  it('shows no answer for a vault whose removals are not paused', async () => {
    const { wrapper } = await render({ vaults: [createVault({ state: 'connected' })] })

    expect(findButton(wrapper, 'Remove the files')).toBeUndefined()
  })
})
