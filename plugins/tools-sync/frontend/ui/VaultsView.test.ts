import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { RemoteVault, Vault, VaultsClient } from '../api/vaults'
import VaultsView from './VaultsView.vue'
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
      vaults: setup.remote ?? [
        { remoteId: 'aa11', name: 'Work', isConnected: false, sizeBytes: 3_670_016 },
      ],
    }),
  ),
  getConnectPreview: vi.fn<VaultsClient['getConnectPreview']>(() =>
    Promise.resolve({ differCount: setup.differCount ?? 0 }),
  ),
  resolveDeletions: vi.fn<VaultsClient['resolveDeletions']>(() => Promise.resolve()),
  connectVault: vi.fn<VaultsClient['connectVault']>(() => Promise.resolve()),
  stopSyncingVault: vi.fn<VaultsClient['stopSyncingVault']>(() => Promise.resolve()),
  deleteCloudVault: vi.fn<VaultsClient['deleteCloudVault']>(() => Promise.resolve()),
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
  beforeEach(async () => {
    const empty = await render({ vaults: [], remote: [] })
    empty.wrapper.unmount()
  })

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

  it('lists a vault of the cloud that is not synced here with its size, after the synced ones', async () => {
    const { wrapper } = await render({
      vaults: [createVault({ id: 'b', name: 'Synced', state: 'connected', remoteId: 'bb22' })],
      remote: [
        { remoteId: 'bb22', name: 'Synced', isConnected: true, sizeBytes: 1_048_576 },
        { remoteId: 'aa11', name: 'Work', isConnected: false, sizeBytes: 3_670_016 },
      ],
    })

    const rows = wrapper.findAll('li').map((one) => one.text())
    expect(rows).toHaveLength(2)
    expect(rows[0]).toContain('Up to date')
    expect(rows[0]).toContain('1 MB in the cloud')
    expect(rows[1]).toContain('Work')
    expect(rows[1]).toContain('Not syncing')
    expect(rows[1]).toContain('3.5 MB in the cloud')
  })

  it('shows the state word and the size of a vault in one line', async () => {
    const { wrapper } = await render({
      vaults: [createVault({ id: 'b', name: 'Synced', state: 'connected', remoteId: 'bb22' })],
      remote: [{ remoteId: 'bb22', name: 'Synced', isConnected: true, sizeBytes: 1_048_576 }],
    })

    const line = wrapper.get('li').get('[role="status"]').text()
    expect(line.startsWith('Up to date')).toBe(true)
    expect(line).toContain('1 MB in the cloud')
  })

  it('shows one placeholder row without text on the first load, then the rows', async () => {
    const client = createClient({
      vaults: [createVault({ name: 'Synced', state: 'connected' })],
      remote: [],
    })
    const wrapper = mount(VaultsView, { props: { client, now: () => NOW } })

    expect(wrapper.findAll('li')).toHaveLength(1)
    expect(wrapper.get('ul').attributes('aria-busy')).toBe('true')
    expect(wrapper.get('ul').text()).toBe('')

    await flushPromises()

    expect(wrapper.findAll('li')).toHaveLength(1)
    expect(wrapper.get('ul').attributes('aria-busy')).toBeUndefined()
    expect(wrapper.text()).toContain('Synced')
  })

  it('shows as many placeholder rows as the last list had, from one to five', async () => {
    const many = (count: number) =>
      Array.from({ length: count }, (_, index) =>
        createVault({ id: `v${index}`, name: `Vault ${index}`, state: 'connected' }),
      )
    const first = await render({ vaults: many(3), remote: [] })
    first.wrapper.unmount()

    const second = mount(VaultsView, {
      props: { client: createClient({ vaults: many(3), remote: [] }), now: () => NOW },
    })
    expect(second.findAll('li')).toHaveLength(3)
    await flushPromises()
    expect(second.findAll('li')).toHaveLength(3)
    second.unmount()

    const big = await render({ vaults: many(8), remote: [] })
    big.wrapper.unmount()
    const third = mount(VaultsView, {
      props: { client: createClient({ vaults: many(8), remote: [] }), now: () => NOW },
    })
    expect(third.findAll('li')).toHaveLength(5)
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
    client.listVaults.mockRejectedValueOnce(new ReportedError('boom'))
    const wrapper = mount(VaultsView, { props: { client, now: () => NOW } })
    await flushPromises()
    expect(wrapper.text()).toContain('boom')

    await findButton(wrapper, 'Try again')?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('boom')
  })
})
