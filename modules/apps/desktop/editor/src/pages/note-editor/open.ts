/**
 * Tab state factory for an open note tab.
 */
import { computed } from 'vue'
import { pointsAtNote } from '@numen/ui'
import type { WindowHandle } from '@/entities/tab'
import type { FileOpeners } from '@/entities/tab'
import type { noteChanges } from './changes'
import type { noteKeyboard } from './keyboard'
import type { openNotes } from '@/entities/note'
import type { noteTitles } from './titles'
import type { NoteTabDeps, NoteTabState } from './types'

type Notes = ReturnType<typeof openNotes>
type NoteChanges = ReturnType<typeof noteChanges>
type NoteKeyboard = ReturnType<typeof noteKeyboard>
type NoteTitles = ReturnType<typeof noteTitles>

/** Creates the reactive state for an open note tab. */
export function createNoteTab(
  id: string,
  notes: Notes,
  changes: NoteChanges,
  keyboard: NoteKeyboard,
  vault: NoteTabDeps,
  names: NoteTitles,
  handle: WindowHandle,
  puts: FileOpeners,
): NoteTabState {
  const closeTab = (tab: string) => {
    keyboard.drops(id)
    changes.shut(notes.where(id))
    void notes.shut(id).then((gone) => {
      if (!gone) return
      names.forgets(id)
      handle.closes(tab)
    })
  }

  const followLink = (url: string) => {
    if (!pointsAtNote(url)) return
    const from = notes.where(id)
    void vault.resolve(from, [url]).then((landed) => {
      const path = landed.get(url)
      if (path) void puts.opens(path, '', 'beside')
    })
  }

  return {
    id,
    shown: computed(() => notes.shown(id)),
    errorMessage: computed(() => notes.getErrorMessage(id)),
    change: computed(() => changes.shown(notes.where(id))),
    updateBody: (body: string) => notes.setBody(id, body),
    save: () => notes.save(id),
    keepMine: () => notes.keep(id),
    takeFile: () => notes.take(id),
    setEditor: (editor: unknown) => keyboard.drew(id, editor),
    measure: () => keyboard.measure(id),
    followLink,
    close: closeTab,
  }
}
