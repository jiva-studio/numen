/**
 * Vaults listing, active vault tracking, and coverage monitoring.
 */
import { computed, ref, type Ref } from 'vue'
import type { VaultList } from '@/entities/vault'
import type { IndexCoverage } from '@/features/command-palette'
import type { MessageLog } from '@/shared/notices/messages'
import type { VaultRef } from '@/entities/command'
import { WINDOW_WORDS } from './words'
import { getVaultList } from './vaultList'

export interface VaultsDeps {
  log: MessageLog
  chunks: Ref<number>
  embedded: Ref<number>
  isEmbedding: Ref<boolean>
}

export function useVaults({ log, chunks, embedded, isEmbedding }: VaultsDeps) {
  const reload = () => globalThis.location.reload()
  const items = ref<VaultList>({ vaults: [], showing: '' })

  /**
   * The vault being shown, which is the one the list reports as showing. It is
   * derived from the list so the two cannot disagree about which vault is open.
   */
  const displayedItems = computed<VaultRef>(() => {
    const one = items.value.vaults.find((v) => v.id === items.value.showing)
    return one ? { id: one.id, name: one.name } : { id: '', name: '' }
  })

  /** One vault of the list, with its name replaced by the given name. */
  const setVaultName = (vault: VaultRef) => {
    items.value = {
      ...items.value,
      vaults: items.value.vaults.map((one) =>
        one.id === vault.id ? { ...one, name: vault.name } : one,
      ),
    }
  }
  const unlisted = log.getWriter('listed')

  const loadVaults = async () => {
    try {
      items.value = await getVaultList()
    } catch {
      // The layout the window opens with is the one the list's answer decides.
      unlisted(WINDOW_WORDS.unlistedVaults, 'error')
    }
  }

  const coverage = (): IndexCoverage => ({
    chunks: chunks.value,
    embedded: embedded.value,
    isEmbedding: isEmbedding.value,
  })

  return {
    reload,
    activeVault: displayedItems,
    setVaultName,
    vaultList: items,
    loadVaults,
    coverage,
  }
}
