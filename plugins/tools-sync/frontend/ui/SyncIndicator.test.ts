import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Vault, VaultsClient } from '../api/vaults'
import { createSyncState } from '../model/sync'
import SyncIndicator from './SyncIndicator.vue'
import { NO_TRANSFER } from './testing/vaultFixture'

const createVault = (patch: Partial<Vault> = {}): Vault => ({
  id: 'Test',
  name: 'Test',
  path: '/vaults/Test',
  state: 'connected',
  lastSyncedAt: '',
  conflictCount: 0,
  remoteId: '',
  ...NO_TRANSFER,
  ...patch,
})

const createClient = (vaults: Vault[]) => ({
  getStatus: vi.fn<VaultsClient['getStatus']>(() =>
    Promise.resolve({ hasIdentity: true, isReachable: true, items: [] }),
  ),
  listVaults: vi.fn<VaultsClient['listVaults']>(() => Promise.resolve({ vaults })),
})

const render = async (vaults: Vault[]) => {
  const state = createSyncState(createClient(vaults))
  state.start()
  const wrapper = mount(SyncIndicator, { props: { state }, attachTo: document.body })
  await flushPromises()
  return { wrapper, state }
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SyncIndicator', () => {
  it('shows the percentage of the bytes of all syncing vaults', async () => {
    const { wrapper, state } = await render([
      createVault({ id: 'a', syncing: true, syncedBytes: 100, totalBytes: 200 }),
      createVault({ id: 'b', syncing: true, syncedBytes: 260, totalBytes: 400 }),
    ])

    expect(wrapper.text()).toContain('Syncing 60%')
    state.stop()
  })

  it('shows the synced state when no vault syncs', async () => {
    const { wrapper, state } = await render([createVault()])

    expect(wrapper.text()).toContain('Synced')
    expect(wrapper.text()).not.toContain('%')
    state.stop()
  })

  it('lists every synced vault with its state and progress when clicked', async () => {
    const { wrapper, state } = await render([
      createVault({ id: 'a', name: 'Busy', syncing: true, syncedBytes: 50, totalBytes: 200 }),
      createVault({ id: 'b', name: 'Calm' }),
    ])
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)

    await wrapper.find('button').trigger('click')

    const rows = wrapper.findAll('[role="dialog"] li')
    expect(rows).toHaveLength(2)
    expect(rows[0]?.text()).toContain('Busy')
    expect(rows[0]?.find('[role="progressbar"]').attributes('aria-valuenow')).toBe('25')
    expect(rows[1]?.text()).toContain('Up to date')
    state.stop()
  })
})
