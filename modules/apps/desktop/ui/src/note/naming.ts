/**
 * What each open note is called.
 *
 * A note is called what the vault calls it, and a heading written into a note
 * is that note's title. So a note is asked what it is called again once what
 * was typed into it has landed.
 */
import { ref, watch } from 'vue'
import type { editing } from './editing'

/** The notes of the whole window, as far as this reads them. */
type Notes = ReturnType<typeof editing>

/** What a note is asked to be called, as the vault last said it. */
export interface Called {
  neighbourhood(path: string): Promise<{ focus?: { title?: string } | undefined }>
}

export function naming(vault: Called, notes: Notes) {
  /** What each note is called, as the vault last said it. */
  const titles = ref<ReadonlyMap<string, string>>(new Map())

  const calls = (path: string, name: string): void => {
    titles.value = new Map(titles.value).set(path, name)
  }

  const forgets = (path: string): void => {
    const rest = new Map(titles.value)
    rest.delete(path)
    titles.value = rest
  }

  /** A vault that cannot answer leaves the note under the name it had. */
  const asks = async (path: string): Promise<void> => {
    try {
      const said = (await vault.neighbourhood(path)).focus?.title
      if (said) calls(path, said)
    } catch {
      return
    }
  }

  watch(
    () => notes.all().filter((path) => notes.shown(path).state === 'clean'),
    (settled, before) => {
      for (const path of settled) {
        if (!before?.includes(path)) void asks(path)
      }
    },
    { deep: true },
  )

  /** What one note is called, and the path it is filed at while nothing else is. */
  const called = (path: string): string => titles.value.get(path) ?? path

  return { titles, calls, forgets, called }
}
