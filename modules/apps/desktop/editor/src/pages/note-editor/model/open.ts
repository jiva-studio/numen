/**
 * Tab state factory for an open note tab.
 */
import { computed } from 'vue'
import { isNoteAddress } from '@numen/ui'
import type { WindowHandle } from '@/entities/tab'
import type { FileOpeners } from '@/entities/tab'
import type { createNoteChanges } from './changes'
import type { createNoteKeyboard } from './keyboard'
import type { openNotes } from '@/entities/note'
import type { createNoteTitles } from './titles'
import type { NoteTabDeps, NoteTabState } from '../types'

type Notes = ReturnType<typeof openNotes>
type NoteChanges = ReturnType<typeof createNoteChanges>
type NoteKeyboard = ReturnType<typeof createNoteKeyboard>
type NoteTitles = ReturnType<typeof createNoteTitles>

/** Creates the reactive state for an open note tab. */
export function createNoteTab(
  id: string,
  notes: Notes,
  changes: NoteChanges,
  keyboard: NoteKeyboard,
  vault: NoteTabDeps,
  names: NoteTitles,
  handle: WindowHandle,
  tabOpeners: FileOpeners,
): NoteTabState {
  const closeTab = (tab: string) => {
    keyboard.cancelFocusRequest(id)
    changes.closeNote(notes.getPath(id))
    void notes.close(id).then((gone) => {
      if (!gone) return
      names.forgetTab(id)
      handle.closeTab(tab)
    })
  }

  const followLink = (url: string) => {
    if (!isNoteAddress(url)) return
    const from = notes.getPath(id)
    void vault.resolve(from, [url]).then((landed) => {
      const path = landed.get(url)
      if (path) void tabOpeners.openFile(path, '', 'beside')
    })
  }

  return {
    id,
    note: computed(() => notes.getOpenNote(id)),
    errorMessage: computed(() => notes.getErrorMessage(id)),
    change: computed(() => changes.getChange(notes.getPath(id))),
    updateBody: (body: string) => notes.setBody(id, body),
    save: () => notes.save(id),
    keepMine: () => notes.keep(id),
    takeFile: () => notes.take(id),
    setEditor: (editor: unknown) => keyboard.setEditor(id, editor),
    measure: () => keyboard.measure(id),
    followLink,
    close: closeTab,
  }
}
