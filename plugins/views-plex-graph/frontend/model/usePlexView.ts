/**
 * The state of one plex: the neighbourhood it shows and how it navigates.
 *
 * A plex tab holds one of these and nothing else: the vault is read once for
 * the whole window, and notifies every plex when to query again.
 */
import { ref } from 'vue'
import { formatErrorMessage, getRenamedPath } from '@numen/plugin-sdk'
import { createRequestGuard } from '@numen/ui'
import { areNeighbourhoodsEqual } from '../lib/picture'
import type { Neighbourhood, PathRename } from '../types'

/** The one query a plex makes of the vault: the neighbours of a note. */
export interface Neighbours {
  neighbourhood(path: string): Promise<Neighbourhood>
}

export type PlexView = ReturnType<typeof usePlexView>

export function usePlexView(core: Neighbours) {
  const neighbourhood = ref<Neighbourhood | null>(null)
  /** The note this plex shows, as requested. */
  const here = ref('')
  /** What this plex could not show, in words the window sets up for it. */
  const error = ref('')
  /**
   * The note a vault change requests again: one whose response has not arrived,
   * or one the vault did not hold yet. Empty when there is none, and the plex
   * then requests `here`.
   */
  const nextPath = ref('')

  /**
   * Whether the first neighbourhood is still on its way. An empty plex and one
   */
  const isLoading = ref(true)

  /** Two responses can be in flight — a click while a change is being followed. */
  const guard = createRequestGuard()

  async function go(path: string) {
    if (!guard.open()) return
    const mine = guard.acquireRequest()
    nextPath.value = path
    try {
      const answer = await core.neighbourhood(path)
      if (!mine.isCurrent) return
      if (!answer.focus.path) {
        error.value = `${path} is not in the vault`
        return
      }
      error.value = ''
      nextPath.value = ''
      here.value = path
      if (!areNeighbourhoodsEqual(neighbourhood.value, answer)) neighbourhood.value = answer
    } catch (caughtError) {
      if (!mine.isCurrent) return
      nextPath.value = ''
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

  return { neighbourhood, here, error, nextPath, isLoading, go, followMoves, close }
}
