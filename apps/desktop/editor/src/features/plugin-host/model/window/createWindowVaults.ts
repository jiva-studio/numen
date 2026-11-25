import { computed, watch, type Ref } from 'vue'
import type { VaultsHost, VaultsState } from '@numen/plugin-loader'
import type { VaultRef } from '@/shared/model/command'
import { formatVaultSize, getCloudRowId, type CloudVault, type VaultList } from '@/entities/vault'

interface WindowVaultsDeps {
  readonly vaultList: Readonly<Ref<VaultList>>
  /** The cloud vaults that are not on this device. */
  readonly cloud?: Readonly<Ref<readonly CloudVault[]>> | undefined
  readonly openVault: (vault: VaultRef) => Promise<void>
}

export function createWindowVaults(deps: WindowVaultsDeps): VaultsHost {
  const state = computed<VaultsState>(() => {
    const { vaults: listed, showing } = deps.vaultList.value
    const current = listed.find((one) => one.id === showing)
    const cloud = (deps.cloud?.value ?? []).map((one) => ({
      id: getCloudRowId(one.remoteId),
      name: one.name,
      detail: formatVaultSize(one.sizeBytes),
    }))
    return {
      vaults: listed.map(({ id, name }) => ({ id, name })),
      current: current ? { id: current.id, name: current.name } : null,
      ...(cloud.length > 0 ? { cloud } : {}),
    }
  })

  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    switchTo: async (id) => {
      const one = deps.vaultList.value.vaults.find((vault) => vault.id === id)
      if (one) return deps.openVault({ id: one.id, name: one.name })
      const remote = state.value.cloud?.find((vault) => vault.id === id)
      if (remote) await deps.openVault({ id: remote.id, name: remote.name })
    },
  }
}
