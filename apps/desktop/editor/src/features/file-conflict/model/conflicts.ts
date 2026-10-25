/**
 * The notes whose file moved past what was read, told to the flush.
 *
 * A conflict is raised while the tab stands stale and dropped when it
 * stops, so the window waits on exactly what is still to be settled.
 */
import { watch } from 'vue'
import { parseFileConflict } from '@numen/plugin-sdk'
import type { Conflict } from './flush'

/** The notes of a window, each under the identity its tab opened under. */
export interface Notes {
  getOpenIds(): readonly string[]
  /** The word the screen holding that note tells its state by. */
  getOpenNote(id: string): { state: string }
  keep(id: string): void
  take(id: string): void
}

/** Where a conflict is raised, so the flush waits until it is settled. */
export interface ConflictRaiser {
  raise(one: Conflict): () => void
}

/** Raises the conflicts of the notes, and answers what stops raising them and drops those still raised. */
export function raiseConflicts(notes: Notes, raiser: ConflictRaiser): () => void {
  const raised = new Map<string, () => void>()

  const stopWatching = watch(
    () =>
      notes.getOpenIds().filter((id) => {
        const c = parseFileConflict(notes.getOpenNote(id).state)
        return c === 'stale'
      }),
    (stale) => {
      for (const id of stale) {
        if (raised.has(id)) continue
        raised.set(
          id,
          raiser.raise({
            note: id,
            keep: async () => notes.keep(id),
            take: async () => notes.take(id),
          }),
        )
      }
      for (const [id, drop] of raised) {
        if (stale.includes(id)) continue
        drop()
        raised.delete(id)
      }
    },
  )

  return () => {
    stopWatching()
    for (const drop of raised.values()) drop()
    raised.clear()
  }
}
