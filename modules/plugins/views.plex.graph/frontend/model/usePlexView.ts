/**
 * What one plex is showing, and how it travels.
 *
 * A plex tab holds one of these, and holds nothing else: the vault is read once
 * for the whole window, and tells every plex when to ask again.
 */
import { ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { createRequestGuard, getRenamedPath } from '@numen/plugin-sdk'
import { areNeighbourhoodsEqual } from '../lib/picture'
import type { Neighbourhood, PathRename } from '../types'

/** The one question a plex asks of the vault: what is around a note. */
export interface Neighbours {
  neighbourhood(path: string): Promise<Neighbourhood>
}

export type PlexView = ReturnType<typeof usePlexView>

export function usePlexView(core: Neighbours) {
  const neighbourhood = ref<Neighbourhood | null>(null)
  /**
   * The note this plex is showing, as it asked for it.
   */
  const here = ref('')
  /** What this plex could not show, in words the window sets up for it. */
  const error = ref('')

  /**
   * Whether the first neighbourhood is still on its way. An empty plex and one
   */
  const isLoading = ref(true)

  /** Two responses can be in flight — a click while a change is being followed. */
  const guard = createRequestGuard()

  async function go(path: string) {
    if (!guard.open()) return
    const mine = guard.acquireRequest()
    try {
      const answer = await core.neighbourhood(path)
      if (!mine.isCurrent) return
      if (!answer.focus.path) {
        error.value = `${path} is not in the vault`
        return
      }
      error.value = ''
      here.value = path
      if (!areNeighbourhoodsEqual(neighbourhood.value, answer)) neighbourhood.value = answer
    } catch (caughtError) {
      if (!mine.isCurrent) return
      error.value = formatErrorMessage(caughtError)
    } finally {
      if (mine.isCurrent) isLoading.value = false
    }
  }

  const followMoves = (renames: readonly PathRename[]) => {
    const to = getRenamedPath(renames, here.value)
    if (!to) return
    here.value = to
    guard.cancel()
  }

  const close = () => {
    guard.close()
  }

  return { neighbourhood, here, error, isLoading, go, followMoves, close }
}

