/**
 * Vaults listing, active vault tracking, and coverage monitoring.
 */
import { computed, ref, type Ref } from 'vue'
import type { VaultList } from '../lib/vault'
import type { IndexCoverage } from '@/shared/model/notices/coverage'
import type { MessageLog } from '@/shared/model/notices/messages'
import { VAULT_WORDS } from '../words'

/** A vault as the commands and the target of the window name it. */
export interface VaultRef {
  readonly id: string
  readonly name: string
}

export interface VaultsDeps {
  listVaults: () => Promise<VaultList>
  log: MessageLog
  chunks: Ref<number>
  embedded: Ref<number>
  isEmbedding: Ref<boolean>
}

export function useVaults({ listVaults, log, chunks, embedded, isEmbedding }: VaultsDeps) {
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
      items.value = await listVaults()
    } catch {
      // The layout the window opens with is the one the list's answer decides.
      unlisted(VAULT_WORDS.unlisted, 'error')
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
