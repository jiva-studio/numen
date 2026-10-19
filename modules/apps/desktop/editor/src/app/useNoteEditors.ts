/**
 * Editing, conflict handling, and flushing for open notes.
 */
import { openNotes, createNoteWriter } from '@/entities/note'
import { raiseConflicts, useFileFlush } from '@/features/file-conflict'
import { createNotes, type SearchDeps, type Store } from '@/features/command-palette'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'
import type { MessageLog } from '@/shared/notices/messages'
import type { FileOpeners } from '@/entities/tab'
import type { useWindowTabs } from '@/entities/tab'

export interface NoteEditorsDeps {
  core: NotePort & SearchDeps & Pick<VaultPort, 'watchQuit' | 'reportFlush'>
  log: MessageLog
  tabOpeners: FileOpeners
  held: ReturnType<typeof useWindowTabs>
}

function createNullChanges() {
  return {
    close: () => {},
    reportChange: () => {},
    reportNoteChange: () => {},
  }
}

function createNoted(held: ReturnType<typeof useWindowTabs>) {
  return {
    measureAll: () => {
      for (const tab of held.tabs.value) {
        const windowTab = held.getTab(tab.id)
        const state = windowTab?.state as
          { measure?: () => void; editor?: { measure?: () => void } } | undefined
        if (state?.measure) {
          state.measure()
        } else if (state?.editor?.measure) {
          state.editor.measure()
        }
      }
    },
  }
}

function createNotedStore(notes: ReturnType<typeof openNotes>): Store {
  return {
    has: (id) => notes.has(id),
    getPath: (id) => notes.getPath(id),
    getTitle: (id) => notes.getPath(id),
    asking: (id) => notes.stale(id) !== null,
    flushPending: (id) => notes.flushPending(id),
    close: (id) => void notes.close(id),
    getTabAt: () => null,
  }
}

export function useNoteEditors({ core, log, tabOpeners, held }: NoteEditorsDeps) {
  const changes = createNullChanges()
  const notes = openNotes(core, { onReplaced: changes.reportNoteChange })
  const making = createNoteWriter(core, log.getWriter('made'))
  const noted = createNoted(held)

  const fileFlush = useFileFlush(core)
  fileFlush.addHandler(notes.flush)

  raiseConflicts(notes, fileFlush)

  const notedStore = createNotedStore(notes)
  const stores: readonly Store[] = [notedStore]
  const reached = createNotes(stores, tabOpeners)
  const getTitle = (id: string): string => stores.find((one) => one.has(id))?.getTitle(id) ?? ''

  const close = () => {
    changes.close()
    fileFlush.close()
  }

  return {
    changes,
    notes,
    making,
    noted,
    fileFlush,
    stores,
    reached,
    getTitle,
    close,
  }
}
