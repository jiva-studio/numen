import { computed, ref } from 'vue'
import { getPlexLayout } from '../lib/plexLayout'
import { ENTROPY_PATH } from '../lib/notes'
import type { Vault } from './vault'

/** The note the plex is centered on, and the notes drawn around it. */
export function createPlex(vault: Vault) {
  const here = ref(ENTROPY_PATH)
  const neighbourhood = computed(() => vault.getNeighbours(here.value))
  const layout = computed(() => getPlexLayout(neighbourhood.value))

  const go = (path: string) => {
    here.value = vault.getNote(path) ? path : ENTROPY_PATH
  }

  vault.onRename((from, to) => {
    if (here.value === from) here.value = to
  })

  return { here, neighbourhood, layout, go }
}

export type Plex = ReturnType<typeof createPlex>
