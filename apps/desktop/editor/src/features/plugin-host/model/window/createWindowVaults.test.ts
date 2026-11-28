/**
 * The vaults the window gives its plugins: the vaults of the installation as the welcome screen
 * lists them, the one the window shows, and the switch to another.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { VaultsState } from '@numen/plugin-loader'
import { COMMAND, type VaultRef } from '@/shared/model/command'
import type { VaultList } from '@/entities/vault'
import { createWindowVaults } from './createWindowVaults'

const vault = (id: string, name: string) => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: false,
  isReadOnly: false,
})

const LIST: VaultList = {
  vaults: [
    vault('orchard', 'Orchard notes'),
    vault('greenhouse', 'Greenhouse log'),
    vault('cellar', 'Cellar book'),
  ],
  showing: 'greenhouse',
}

function setup(list: VaultList = LIST) {
  const vaultList = ref(list)
  const openVault = vi.fn<(vault: VaultRef) => Promise<void>>(async () => undefined)
  const vaults = createWindowVaults({ vaultList, openVault })
  return { vaults, vaultList, openVault }
}

describe('the vaults of the window', () => {
  it('list only the vaults of the installation, in the order of the vault list', () => {
    expect(setup().vaults.getState().vaults).toStrictEqual([
      { id: 'orchard', name: 'Orchard notes' },
      { id: 'greenhouse', name: 'Greenhouse log' },
      { id: 'cellar', name: 'Cellar book' },
    ])
  })

  it('name the vault the window shows as the current one', () => {
    expect(setup().vaults.getState().current).toStrictEqual({
      id: 'greenhouse',
      name: 'Greenhouse log',
    })
  })

  it('name no current vault while the window shows none', () => {
    expect(setup({ ...LIST, showing: '' }).vaults.getState().current).toBeNull()
  })

  it('call a watcher at once and again when the vault list changes, until it is disposed', async () => {
    const { vaults, vaultList } = setup()
    const states: VaultsState[] = []
    const watch = vaults.watch((state) => states.push(state))

    vaultList.value = { vaults: [vault('cellar', 'Cellar book')], showing: 'cellar' }
    await nextTick()
    watch.dispose()
    vaultList.value = LIST
    await nextTick()

    expect(states.map((one) => one.current?.id)).toStrictEqual(['greenhouse', 'cellar'])
    expect(states[1]?.vaults.map((one) => one.id)).toStrictEqual(['cellar'])
  })
})

describe('switching the vault of the window', () => {
  it('opens a vault of the list as the welcome screen does, with the id and the name of the vault', async () => {
    const { vaults, openVault } = setup()

    await vaults.switchTo('cellar')

    expect(openVault.mock.calls).toStrictEqual([[{ id: 'cellar', name: 'Cellar book' }]])
  })

  it('opens nothing for the id of the command that creates a vault', async () => {
    const { vaults, openVault } = setup()

    await vaults.switchTo(COMMAND.createVault)

    expect(openVault).not.toHaveBeenCalled()
  })

  it('opens nothing for an id the list does not hold', async () => {
    const { vaults, openVault } = setup()

    await vaults.switchTo('lost')

    expect(openVault).not.toHaveBeenCalled()
  })
})

describe('the cloud vaults of the window', () => {
  const HARE = { remoteId: 'r1', name: 'Hare Krishna', sizeBytes: 2040109466, vaultId: '' }

  it('list them with the size under the name, and none where there is none', () => {
    const vaultList = ref(LIST)
    const cloud = ref([HARE])
    const vaults = createWindowVaults({ vaultList, cloud, openVault: async () => undefined })

    expect(vaults.getState().cloud).toStrictEqual([
      { id: 'cloud:r1', name: 'Hare Krishna', detail: '1.9 GB' },
    ])

    cloud.value = []
    expect(vaults.getState().cloud).toBeUndefined()
  })

  it('start getting the vault when its row is switched to', async () => {
    const openVault = vi.fn<(vault: VaultRef) => Promise<void>>(async () => undefined)
    const vaults = createWindowVaults({
      vaultList: ref(LIST),
      cloud: ref([HARE]),
      openVault,
    })

    await vaults.switchTo('cloud:r1')

    expect(openVault).toHaveBeenCalledWith({ id: 'cloud:r1', name: 'Hare Krishna' })
  })
})
