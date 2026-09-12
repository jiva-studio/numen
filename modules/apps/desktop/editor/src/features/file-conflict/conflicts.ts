/**
 * The notes whose file moved past what was read, told to the flush.
 *
 * A conflict is raised while the tab stands stale and dropped when it
 * stops, so the window waits on exactly what is still to be settled.
 */
import { watch } from 'vue'
import { conflictIn, type Conflict } from './flush'

/** The notes of a window, each under the identity its tab opened under. */
export interface Notes {
  all(): readonly string[]
  /** The word the screen holding that note tells its state by. */
  getOpenNote(id: string): { state: string }
  keep(id: string): void
  take(id: string): void
}

/** Where a conflict is raised, so the flush waits until it is settled. */
export interface ConflictRaiser {
  raise(one: Conflict): () => void
}

export function raiseConflicts(notes: Notes, going: ConflictRaiser) {
  const raised = new Map<string, () => void>()

  watch(
    () =>
      notes.all().filter((id) => {
        const c = conflictIn(notes.getOpenNote(id).state)
        return c === 'stale'
      }),
    (stale) => {
      for (const id of stale) {
        if (raised.has(id)) continue
        raised.set(
          id,
          going.raise({
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
    { deep: true },
  )
}
