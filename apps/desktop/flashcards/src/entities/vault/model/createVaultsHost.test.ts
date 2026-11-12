/**
 * The vaults the flashcards window gives its plugins: the vaults it lists to choose from, the vault
 * of the tab it shows, and a switch that opens the tab on another vault.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, shallowRef } from 'vue'
import type { VaultsState } from '@numen/plugin-loader'
import type { Vault } from '../lib/vault'
import { createVaultsHost } from './createVaultsHost'

const ROWS: readonly Vault[] = [
  { id: 'orchard', name: 'Orchard notes', path: '/vaults/Orchard notes' },
  { id: 'cellar', name: 'Cellar book', path: '/vaults/Cellar book' },
]

function setup(vault: string | null = null) {
  const list = shallowRef<readonly Vault[]>(ROWS)
  const tabVault = shallowRef<string | null>(vault)
  const openTab = vi.fn<(id: string) => void>()
  const vaults = createVaultsHost({ list, getTabVault: () => tabVault.value, openTab })
  return { vaults, list, tabVault, openTab }
}

describe('the vaults of the flashcards window', () => {
  it('list the vaults the window lists, by id and name', () => {
    expect(setup().vaults.getState().vaults).toStrictEqual([
      { id: 'orchard', name: 'Orchard notes' },
      { id: 'cellar', name: 'Cellar book' },
    ])
  })

  it('name no current vault while no tab is open', () => {
    expect(setup().vaults.getState().current).toBeNull()
  })

  it('name the vault of the open tab as the current one', () => {
    expect(setup('cellar').vaults.getState().current).toStrictEqual({
      id: 'cellar',
      name: 'Cellar book',
    })
  })

  it('call a watcher at once and again when a tab opens, until it is disposed', async () => {
    const { vaults, tabVault } = setup()
    const states: VaultsState[] = []
    const watch = vaults.watch((state) => states.push(state))

    tabVault.value = 'orchard'
    await nextTick()
    watch.dispose()
    tabVault.value = 'cellar'
    await nextTick()

    expect(states.map((one) => one.current?.id ?? null)).toStrictEqual([null, 'orchard'])
  })

  it('open the tab on the vault switched to', async () => {
    const { vaults, openTab } = setup('orchard')

    await vaults.switchTo('cellar')

    expect(openTab.mock.calls).toStrictEqual([['cellar']])
  })

  it('open nothing for an id the list does not hold', async () => {
    const { vaults, openTab } = setup()

    await vaults.switchTo('lost')

    expect(openTab).not.toHaveBeenCalled()
  })
})
