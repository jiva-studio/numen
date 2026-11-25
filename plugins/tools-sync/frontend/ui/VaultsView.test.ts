import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { RemoteVault, Vault, VaultsClient } from '../api/vaults'
import VaultsView from './VaultsView.vue'
import { ReportedError } from '@numen/plugin-sdk'
import { NO_TRANSFER } from './testing/vaultFixture'

const NOW = new Date('2026-11-23T12:00:00Z')

const createVault = (patch: Partial<Vault> = {}): Vault => ({
  id: 'Test',
  name: 'Test',
  path: '/vaults/Test',
  state: 'notConnected',
  lastSyncedAt: '',
  conflictCount: 0,
  remoteId: '',
  ...NO_TRANSFER,
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

  it('puts the state word with its qualifier on one line and the facts on another, one unit per fact', async () => {
    const { wrapper } = await render({
      vaults: [
        createVault({
          id: 'b',
          name: 'Synced',
          state: 'connected',
          remoteId: 'bb22',
          lastSyncedAt: '2026-11-23T11:55:00Z',
          onDemandFiles: 151,
          totalBytes: 1_048_576,
        }),
      ],
      remote: [{ remoteId: 'bb22', name: 'Synced', isConnected: true, sizeBytes: 1_048_576 }],
    })

    const label = wrapper.get('li .vault-row__label')
    expect(label.get('.vault-row__name').text()).toBe('Synced')
    const getPieces = (selector: string): string[] =>
      label
        .findAll(selector)
        .flatMap((line) => line.findAll('.vault-row__piece'))
        .map((one) => one.text())
    expect(getPieces('[role="status"]')).toStrictEqual(['Up to date ·', 'Last sync 5 minutes ago'])
    expect(getPieces('.vault-row__line--facts')).toStrictEqual([
      '151 files on demand ·',
      '1 MB in the cloud',
    ])
  })

  it('shows no line of facts for a vault that syncs now', async () => {
    const { wrapper } = await render({
      vaults: [
        createVault({ state: 'connected', syncing: true, syncedBytes: 1, totalBytes: 4_194_304 }),
      ],
      remote: [],
    })

    expect(wrapper.findAll('li .vault-row__line')).toHaveLength(1)
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

    expect(wrapper.text()).toContain('5 of 10 files are removed here')
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

  it('draws the failure as the block of an empty card: a title, the text and one button that retries', async () => {
    const client = createClient()
    client.listVaults.mockRejectedValueOnce(new ReportedError('boom'))
    const wrapper = mount(VaultsView, { props: { client, now: () => NOW } })
    await flushPromises()

    const block = wrapper.get('.empty-block')
    expect(block.get('.empty-block__title').text()).toBe('The vaults did not load')
    expect(block.get('.empty-block__text').text()).toBe('boom')
    expect(block.findAll('button').map((one) => one.text())).toEqual(['Try again'])

    await block.get('button').trigger('click')
    await flushPromises()

    expect(client.listVaults).toHaveBeenCalledTimes(2)
  })

  it('shows a syncing vault with its sizes and a progress ring, and an up-to-date vault with a check and no ring', async () => {
    const { wrapper } = await render({
      vaults: [
        createVault({
          id: 'a',
          name: 'Busy',
          state: 'connected',
          syncing: true,
          syncedBytes: 1_048_576,
          totalBytes: 4_194_304,
        }),
        createVault({ id: 'b', name: 'Calm', state: 'connected' }),
      ],
      remote: [],
    })

    const rows = wrapper.findAll('li')
    expect(rows[0]?.text()).toContain('1 MB of 4 MB')
    const ring = rows[0]?.find('[role="progressbar"]')
    expect(ring?.attributes('aria-valuenow')).toBe('25')
    expect(ring?.find('svg.sync-ring').exists()).toBe(true)
    expect(rows[0]?.find('.vault-row__progress').exists()).toBe(false)
    expect(rows[1]?.text()).toContain('Up to date')
    expect(rows[1]?.find('[role="progressbar"]').exists()).toBe(false)
    expect(rows[1]?.find('.vault-row__icon svg.lucide-check').exists()).toBe(true)
  })

  it('marks the ring indeterminate while the size of a syncing vault is unknown', async () => {
    const { wrapper } = await render({
      vaults: [createVault({ state: 'connected', syncing: true, syncedBytes: 0, totalBytes: 0 })],
      remote: [],
    })

    const ring = wrapper.get('[role="progressbar"]')
    expect(ring.attributes('aria-busy')).toBe('true')
    expect(ring.attributes('aria-valuenow')).toBeUndefined()
  })

  it('lets a conflict win over syncing', async () => {
    const { wrapper } = await render({
      vaults: [createVault({ state: 'connected', syncing: true, conflictCount: 2 })],
      remote: [],
    })

    expect(wrapper.text()).toContain('Needs attention')
    expect(wrapper.find('[role="progressbar"]').exists()).toBe(false)
  })

  it('draws the icon placeholder and both text lines while the vaults load', () => {
    const client = createClient()
    client.getStatus.mockReturnValue(new Promise(() => undefined))
    const wrapper = mount(VaultsView, { props: { client, now: () => NOW } })

    const row = wrapper.find('.vault-row--loading')
    expect(row.find('.vault-row__icon .skeleton').exists()).toBe(true)
    expect(row.findAll('.vault-row__label .skeleton')).toHaveLength(2)
  })
})
