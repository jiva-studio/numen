import { computed, watch, type Ref } from 'vue'
import type { VaultsHost, VaultsState } from '@numen/plugin-loader'
import type { VaultRef } from '@/shared/model/command'
import type { VaultList } from '@/entities/vault'

interface WindowVaultsDeps {
  readonly vaultList: Readonly<Ref<VaultList>>
  readonly openVault: (vault: VaultRef) => Promise<void>
}

export function createWindowVaults(deps: WindowVaultsDeps): VaultsHost {
  const state = computed<VaultsState>(() => {
    const { vaults: listed, showing } = deps.vaultList.value
    const current = listed.find((one) => one.id === showing)
    return {
      vaults: listed.map(({ id, name }) => ({ id, name })),
      current: current ? { id: current.id, name: current.name } : null,
    }
  })

  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    switchTo: async (id) => {
      const one = deps.vaultList.value.vaults.find((vault) => vault.id === id)
      if (one) await deps.openVault({ id: one.id, name: one.name })
    },
  }
}
