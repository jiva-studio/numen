/**
 * The notes whose file moved past what was read, told to the quit.
 *
 * A question is raised while the tab stands overtaken and dropped when it
 * stops, so the window waits on exactly what is still to be answered.
 */
import { watch } from 'vue'
import type { Question } from './leaving'
import type { State } from './note/tab'

/** The notes of a window, each under the identity its tab opened under. */
export interface Notes {
  all(): readonly string[]
  shown(id: string): { state: State }
  keep(id: string): void
  take(id: string): void
}

/** What the window answers when the application says it is going. */
export interface Quit {
  raise(one: Question): () => void
}

export function raising(notes: Notes, going: Quit) {
  const raised = new Map<string, () => void>()

  watch(
    () => notes.all().filter((id) => notes.shown(id).state === 'overtaken'),
    (standing) => {
      for (const id of standing) {
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
        if (standing.includes(id)) continue
        drop()
        raised.delete(id)
      }
    },
    { deep: true },
  )
}
