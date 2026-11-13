/**
 * The state of one plex: the related notes it shows and how it navigates.
 *
 * A plex tab holds one of these and nothing else: the vault is read once for
 * the whole window, and notifies every plex when to query again.
 */
import { ref, watch, type Ref } from 'vue'
import { formatErrorMessage, getRenamedPath } from '@numen/plugin-sdk'
import { createRequestGuard } from '@numen/ui'
import { areNoteGraphsEqual } from '../lib/picture'
import type { NoteGraph, PathRename } from '../types'

/** The one query a plex makes of the vault: the related notes of a note. */
export interface RelatedNoteQueries {
  listRelatedNotes(path: string, depth?: 1 | 2): Promise<NoteGraph>
}

export type PlexView = ReturnType<typeof usePlexView>

/**
 * `depth` is how far the notes are read: 2 also reads the notes related to each related note.
 * The plex reads again when it changes.
 */
export function usePlexView(core: RelatedNoteQueries, depth: Readonly<Ref<1 | 2>> = ref(1)) {
  const noteGraph = ref<NoteGraph | null>(null)
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
   * Whether the first note graph is still on its way, so an empty plex is told from one that has nothing to show.
   */
  const isLoading = ref(true)

  /** Two responses can be in flight — a click while a change is being followed. */
  const guard = createRequestGuard()

  async function go(path: string) {
    if (!guard.open()) return
    const mine = guard.acquireRequest()
    nextPath.value = path
    try {
      const answer = await core.listRelatedNotes(path, depth.value)
      if (!mine.isCurrent) return
      if (!answer.focus.path) {
        error.value = `${path} is not in the vault`
        return
      }
      error.value = ''
      nextPath.value = ''
      here.value = path
      if (!areNoteGraphsEqual(noteGraph.value, answer)) noteGraph.value = answer
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

  const stopWatchingDepth = watch(depth, () => {
    const path = nextPath.value || here.value
    if (path) go(path).catch(() => undefined)
  })

  const close = () => {
    stopWatchingDepth()
    guard.close()
  }

  return { noteGraph, here, error, nextPath, isLoading, go, followMoves, close }
}
