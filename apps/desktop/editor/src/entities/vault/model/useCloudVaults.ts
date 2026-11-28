/**
 * The cloud vaults the window knows of: the last list is kept and shown at once, and a refresh
 * replaces it when the answer comes.
 */
import { computed, shallowRef } from 'vue'
import type { RemoteVaults } from '../api/remoteVaults'
import { selectNewCloudVaults, type CloudVault } from '../lib/cloudVault'

export function useCloudVaults(remote: Pick<RemoteVaults, 'list'>) {
  const known = shallowRef<readonly CloudVault[]>([])
  let isRefreshing = false

  const refresh = async (): Promise<void> => {
    if (isRefreshing) return
    isRefreshing = true
    try {
      known.value = await remote.list()
    } finally {
      isRefreshing = false
    }
  }

  return {
    /** The cloud vaults that are not on this device, as last known. */
    fresh: computed(() => selectNewCloudVaults(known.value)),
    /** Every cloud vault, as last known. */
    all: known,
    refresh,
  }
}

export type CloudVaults = ReturnType<typeof useCloudVaults>
