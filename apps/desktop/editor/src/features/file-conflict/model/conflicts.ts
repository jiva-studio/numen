/**
 * The notes whose file changed after it was read, reported to the flush.
 *
 * A conflict is raised while the tab is out of date and dropped when the tab
 * stops, so the window waits only for conflicts still unresolved.
 */
import { watch } from 'vue'
import type { Conflict } from './flush'

/** The notes of a window, each under the identity its tab opened under. */
export interface Notes {
  getOpenIds(): readonly string[]
  /** The state of the screen that shows that note. */
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
    () => notes.getOpenIds().filter((id) => notes.getOpenNote(id).state === 'stale'),
    (stale) => {
      for (const id of stale) {
        if (raised.has(id)) continue
        raised.set(
          id,
          raiser.raise({
            note: id,
            keep: () => new Promise((resolve) => resolve(notes.keep(id))),
            take: () => new Promise((resolve) => resolve(notes.take(id))),
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
