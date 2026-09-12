/**
 * Vaults listing, active vault tracking, and coverage monitoring.
 */
import { ref, shallowRef, type Ref } from 'vue'
import { running } from '@/shared/artifacts'
import type { ArtifactStates } from '@/shared/artifacts'
import type { VaultList } from '@/shared/vaults'
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
  embedding: Ref<boolean>
}

export function useVaults({ core, words, log, chunks, embedded, embedding }: VaultsDeps) {
  const reload = () => globalThis.location.reload()
  const shown = ref<VaultRef>({ id: '', name: '' })
  const listed = ref<VaultList>({ vaults: [], showing: '' })
  const unlisted = log.under('listed')

  const loadVaults = async () => {
    try {
      const answer = await core.vaults()
      listed.value = answer
      const one = answer.vaults.find((v) => v.id === answer.showing)
      shown.value = one ? { id: one.id, name: one.name } : { id: '', name: '' }
    } catch {
      // The layout the window opens with is the one the list's answer decides.
      unlisted(words.unlistedVaults, 'error')
    }
  }

  const coverage = (): IndexCoverage => ({
    chunks: chunks.value,
    embedded: embedded.value,
    embedding: embedding.value,
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
    listed,
    loadVaults,
    coverage,
    makes,
    loadArtifactStates,
  }
}
