/**
 * Editing, conflict handling, and flushing for open notes.
 */
import { formatError } from '@/shared/api/errors'
import { runInBackground } from '@numen/ui'
import { openNotes, createNoteWriter } from '@/entities/note'
import { reportFlush, watchQuit } from '@/entities/window'
import { raiseConflicts, createFileFlush } from '@/features/file-conflict'
import { createNotes, type Store } from '@/features/command-palette'
import type { MessageLog } from '@/shared/notices/messages'
import type { FileOpeners } from '@/entities/tab'
import type { WindowTabs } from '@/entities/tab'

export interface NoteEditorsDeps {
  log: MessageLog
  tabOpeners: FileOpeners
  windowTabs: WindowTabs
}

function createNullChanges() {
  return {
    // No change listener is attached until the window supplies one.
    close: () => undefined,
    reportChange: () => undefined,
    reportNoteChange: () => undefined,
  }
}

function createNoted(windowTabs: WindowTabs) {
  return {
    measureAll: () => {
      for (const tab of windowTabs.tabs.value) {
        const windowTab = windowTabs.getTab(tab.id)
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

function createNotedStore(
  notes: ReturnType<typeof openNotes>,
  windowTabs: Pick<WindowTabs, 'tabs'>,
  reportError: (error: unknown) => void,
): Store {
  return {
    has: (id) => notes.has(id),
    getPath: (id) => notes.getPath(id),
    getTitle: (id) =>
      windowTabs.tabs.value.find((one) => one.id === id)?.title || notes.getPath(id),
    asking: (id) => notes.stale(id) !== null,
    flushPending: (id) => notes.flushPending(id),
    close: (id) => runInBackground(notes.close(id), reportError),
    getTabAt: (path) => notes.getOpenIds().find((id) => notes.getPath(id) === path) ?? null,
  }
}

export function createNoteEditors({ log, tabOpeners, windowTabs }: NoteEditorsDeps) {
  const changes = createNullChanges()
  const notes = openNotes({ onReplaced: changes.reportNoteChange })
  const making = createNoteWriter(log.getWriter('made'))
  const noted = createNoted(windowTabs)

  const fileFlush = createFileFlush({ watchQuit, reportFlush })
  fileFlush.addHandler(notes.flush)

  raiseConflicts(notes, fileFlush)

  const notedStore = createNotedStore(notes, windowTabs, (error) =>
    log.getWriter('editing')(`A note did not close: ${formatError(error)}`, 'error'),
  )
  const stores: readonly Store[] = [notedStore]
  const reached = createNotes(stores, tabOpeners)
  const getTitle = (id: string): string => stores.find((one) => one.has(id))?.getTitle(id) ?? ''

  const close = () => {
    changes.close()
    fileFlush.close()
  }

  const api = {
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
  return api
}
