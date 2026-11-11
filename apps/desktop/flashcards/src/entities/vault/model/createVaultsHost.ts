/**
 * The vaults the flashcards window gives its plugins: the vaults it lists to choose from, the vault
 * of the tab it shows, and the switch that opens the tab on another vault.
 */
import { computed, watch, type Ref } from 'vue'
import type { VaultsHost, VaultsState } from '@numen/plugin-loader'
import type { Vault } from '../lib/vault'

interface VaultsHostDeps {
  readonly list: Readonly<Ref<readonly Vault[]>>
  readonly getTabVault: () => string | null
  readonly openTab: (vault: string) => void
}

export function createVaultsHost(deps: VaultsHostDeps): VaultsHost {
  const state = computed<VaultsState>(() => {
    const vaults = deps.list.value.map(({ id, name }) => ({ id, name }))
    const tabVault = deps.getTabVault()
    return { vaults, current: vaults.find((one) => one.id === tabVault) ?? null }
  })
  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    switchTo: (id) =>
      new Promise((resolve) => {
        if (deps.list.value.some((one) => one.id === id)) deps.openTab(id)
        resolve()
      }),
  }
}
