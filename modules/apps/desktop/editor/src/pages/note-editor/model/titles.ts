/**
 * What each open note is called.
 *
 * A note is called what the vault calls it, and a heading written into a note
 * is that note's title. So a note is asked what it is called again once what
 * was typed into it has landed.
 */
import { shallowRef, watch } from 'vue'
import type { openNotes } from '@/entities/note'

/** The notes of the whole window, as far as this reads them. */
type Notes = ReturnType<typeof openNotes>

/** What a note is asked to be called, as the vault last said it. */
export interface NoteTitlesDeps {
  neighbourhood(path: string): Promise<{ focus?: { title?: string } | undefined }>
}

/** One settled note: the identity it opened under, and the file it settled at. */
interface SettledNote {
  readonly id: string
  readonly at: string
}

export function noteTitles(vault: NoteTitlesDeps, notes: Notes) {
  /**
   * What each note is called, as the vault last said it, under the identity its
   * tab opened under. A note keeps what it is called wherever its file goes.
   */
  const titles = shallowRef<ReadonlyMap<string, string>>(new Map())

  const setTitle = (id: string, name: string): void => {
    titles.value = new Map(titles.value).set(id, name)
  }

  const forgetTab = (id: string): void => {
    const rest = new Map(titles.value)
    rest.delete(id)
    titles.value = rest
  }

  /**
   * A note is asked about at the file it stands at now. A vault that cannot
   * answer leaves it under the name it had.
   */
  const refreshTitle = async (id: string): Promise<void> => {
    try {
      const said = (await vault.neighbourhood(notes.getPath(id))).focus?.title
      if (said) setTitle(id, said)
    } catch {
      // The tab keeps the name it had, and the next thing that moves the note
      // asks again.
      return
    }
  }

  /** Every note that has settled, each with the file it settled at. */
  const getSettledNotes = (): readonly SettledNote[] =>
    notes
      .getOpenIds()
      .filter((id) => notes.getOpenNote(id).state === 'clean')
      .map((id) => ({ id, at: notes.getPath(id) }))

  /**
   * A note is asked what it is called once what was typed into it has landed,
   * and again once it settles at another file.
   */
  watch(getSettledNotes, (now, before) => {
    for (const one of now) {
      if (!before?.some((was) => was.id === one.id && was.at === one.at)) void refreshTitle(one.id)
    }
  })

  /** What one note is called, and the file it stands at while nothing has named it. */
  const getTitle = (id: string): string => titles.value.get(id) ?? notes.getPath(id)

  return { titles, setTitle, forgetTab, getTitle }
}
