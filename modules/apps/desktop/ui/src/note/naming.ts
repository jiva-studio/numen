/**
 * What each open note is called.
 *
 * A note is called what the vault calls it, and a heading written into a note
 * is that note's title. So a note is asked what it is called again once what
 * was typed into it has landed.
 */
import { shallowRef, watch } from 'vue'
import type { editing } from './editing'

/** The notes of the whole window, as far as this reads them. */
type Notes = ReturnType<typeof editing>

/** What a note is asked to be called, as the vault last said it. */
export interface NamingDeps {
  neighbourhood(path: string): Promise<{ focus?: { title?: string } | undefined }>
}

/** One settled note: the identity it opened under, and the file it settled at. */
interface SettledNote {
  readonly id: string
  readonly at: string
}

export function naming(vault: NamingDeps, notes: Notes) {
  /**
   * What each note is called, as the vault last said it, under the identity its
   * tab opened under. A note keeps what it is called wherever its file goes.
   */
  const titles = shallowRef<ReadonlyMap<string, string>>(new Map())

  const calls = (id: string, name: string): void => {
    titles.value = new Map(titles.value).set(id, name)
  }

  const forgets = (id: string): void => {
    const rest = new Map(titles.value)
    rest.delete(id)
    titles.value = rest
  }

  /**
   * A note is asked about at the file it stands at now. A vault that cannot
   * answer leaves it under the name it had.
   */
  const asks = async (id: string): Promise<void> => {
    try {
      const said = (await vault.neighbourhood(notes.where(id))).focus?.title
      if (said) calls(id, said)
    } catch {
      return
    }
  }

  /** Every note that has settled, each with the file it settled at. */
  const settled = (): readonly SettledNote[] =>
    notes
      .all()
      .filter((id) => notes.shown(id).state === 'clean')
      .map((id) => ({ id, at: notes.where(id) }))

  /**
   * A note is asked what it is called once what was typed into it has landed,
   * and again once it settles at another file.
   */
  watch(settled, (now, before) => {
    for (const one of now) {
      if (!before?.some((was) => was.id === one.id && was.at === one.at)) void asks(one.id)
    }
  })

  /** What one note is called, and the file it stands at while nothing has named it. */
  const called = (id: string): string => titles.value.get(id) ?? notes.where(id)

  return { titles, calls, forgets, called }
}
