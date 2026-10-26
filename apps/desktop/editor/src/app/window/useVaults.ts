/**
 * Vaults listing, active vault tracking, and coverage monitoring.
 */
import { computed, ref, type Ref } from 'vue'
import type { VaultList } from '@/entities/vault'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog } from '@/shared/notices/messages'
import type { VaultRef } from '@/features/command-palette'
import { WORDS } from '@/shared/words'
import type { VaultCore } from '../vault'

type Words = typeof WORDS

export interface VaultsDeps {
  core: VaultCore
  words: Words
  log: MessageLog
  chunks: Ref<number>
  embedded: Ref<number>
  isEmbedding: Ref<boolean>
}

export function useVaults({ core, words, log, chunks, embedded, isEmbedding }: VaultsDeps) {
  const reload = () => globalThis.location.reload()
  const items = ref<VaultList>({ vaults: [], showing: '' })

  /**
   * The vault in front, which is the one the list says it is showing. It is read
   * off the list so that the two cannot disagree about which vault is open.
   */
  const displayedItems = computed<VaultRef>(() => {
    const one = items.value.vaults.find((v) => v.id === items.value.showing)
    return one ? { id: one.id, name: one.name } : { id: '', name: '' }
  })

  /** One vault of the list under the name it was just given. */
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
      items.value = await core.vaults()
    } catch {
      // The layout the window opens with is the one the list's answer decides.
      unlisted(words.unlistedVaults, 'error')
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
