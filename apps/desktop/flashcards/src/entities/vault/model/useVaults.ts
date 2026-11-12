import { onMounted, ref, shallowRef } from 'vue'
import { formatError } from '@/shared/api/errors'
import { listVaults } from '../api/vaults'
import type { Vault } from '../lib/vault'

/**
 * The vaults of the installation, listed once the component mounts. A failure to list is reported
 * to `onError`, and the list stays empty.
 */
export function useVaults(onError: (text: string) => void) {
  const vaults = shallowRef<readonly Vault[]>([])
  const isListing = ref(true)

  onMounted(async () => {
    try {
      vaults.value = await listVaults()
    } catch (error) {
      onError(formatError(error))
    } finally {
      isListing.value = false
    }
  })

  return { vaults, isListing }
}
