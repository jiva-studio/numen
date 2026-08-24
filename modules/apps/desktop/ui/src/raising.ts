/**
 * The notes whose file moved past what was read, told to the quit.
 *
 * A question is raised while the tab stands overtaken and dropped when it
 * stops, so the window waits on exactly what is still to be answered.
 */
import { watch } from 'vue'
import type { Question } from './leaving'
import type { State } from './note/tab'

/** The notes of a window, as far as this reads them. */
export interface Notes {
  all(): readonly string[]
  shown(path: string): { state: State }
  keep(path: string): void
  take(path: string): void
}

/** What the window answers when the application says it is going. */
export interface Quit {
  raise(one: Question): () => void
}

export function raising(notes: Notes, going: Quit) {
  const raised = new Map<string, () => void>()

  watch(
    () => notes.all().filter((path) => notes.shown(path).state === 'overtaken'),
    (standing) => {
      for (const path of standing) {
        if (raised.has(path)) continue
        raised.set(
          path,
          going.raise({
            path,
            keep: async () => notes.keep(path),
            take: async () => notes.take(path),
          }),
        )
      }
      for (const [path, drop] of raised) {
        if (standing.includes(path)) continue
        drop()
        raised.delete(path)
      }
    },
    { deep: true },
  )
}
