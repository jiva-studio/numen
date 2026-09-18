/**
 * Vaults listing, active vault tracking, and coverage monitoring.
 */
import { computed, ref, shallowRef, type Ref } from 'vue'
import { running } from '@/entities/artifact'
import type { ArtifactStates } from '@/entities/artifact'
import type { VaultList } from '@/entities/vault'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog } from '@/shared/notices/messages'
import type { VaultRef } from '@/features/command-palette'
import { WORDS } from '@/shared/words'
import type { VaultCore } from './vault'

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
  const listed = ref<VaultList>({ vaults: [], showing: '' })

  /**
   * The vault in front, which is the one the list says it is showing. It is read
   * off the list so that the two cannot disagree about which vault is open.
   */
  const shown = computed<VaultRef>(() => {
    const one = listed.value.vaults.find((v) => v.id === listed.value.showing)
    return one ? { id: one.id, name: one.name } : { id: '', name: '' }
  })

  /** One vault of the list under the name it was just given. */
  const setVaultName = (vault: VaultRef) => {
    listed.value = {
      ...listed.value,
      vaults: listed.value.vaults.map((one) =>
        one.id === vault.id ? { ...one, name: vault.name } : one,
      ),
    }
  }
  const unlisted = log.getWriter('listed')

  const loadVaults = async () => {
    try {
      listed.value = await core.vaults()
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

  const makes = shallowRef<ReadonlyMap<string, ArtifactStates>>(new Map())

  const loadArtifactStates = async (path: string) => {
    if (!path) return
    try {
      const held = await running.getArtifactStates(path)
      makes.value = new Map(makes.value).set(path, held)
    } catch {
      // A file that cannot be asked about is one nothing is known of, and every
      // command over it is offered as it was before anything could be listed.
      makes.value = new Map(makes.value).set(path, {})
    }
  }

  return {
    reload,
    shown,
    setVaultName,
    listed,
    loadVaults,
    coverage,
    makes,
    loadArtifactStates,
  }
}
