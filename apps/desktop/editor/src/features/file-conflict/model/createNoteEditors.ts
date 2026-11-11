/**
 * Editing, conflict handling, and flushing for open notes.
 */
import { formatError, type Caught } from '@/shared/api/errors'
import { runInBackground } from '@numen/ui'
import { openNotes, createNoteWriter } from '@/entities/note'
import { reportFlush, watchQuit } from '@/entities/window'
import type { WindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/model/notices/messages'
import { raiseConflicts } from './conflicts'
import { createFileFlush } from './flush'

export interface NoteEditorsDeps {
  readonly log: MessageLog
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
}

/** The open notes as the palette reads them: the file of each, the tab it is in, and its title. */
function createNoteStore(
  notes: ReturnType<typeof openNotes>,
  windowTabs: Pick<WindowTabs, 'tabs'>,
  reportError: (error: Caught) => void,
) {
  return {
    has: (id: string) => notes.has(id),
    getPath: (id: string) => notes.getPath(id),
    getTitle: (id: string) =>
      windowTabs.tabs.tabs.value.find((one) => one.id === id)?.title || notes.getPath(id),
    asking: (id: string) => notes.stale(id) !== null,
    flushPending: (id: string) => notes.flushPending(id),
    close: (id: string) => runInBackground(notes.close(id), reportError),
    getTabAt: (path: string) => notes.getOpenIds().find((id) => notes.getPath(id) === path) ?? null,
  }
}

export function createNoteEditors({ log, windowTabs }: NoteEditorsDeps) {
  const notes = openNotes({})
  const making = createNoteWriter(log.getWriter('made'))
  const fileFlush = createFileFlush({ watchQuit, reportFlush })
  fileFlush.addHandler(notes.flush)
  raiseConflicts(notes, fileFlush)

  const store = createNoteStore(notes, windowTabs, (error) =>
    log.getWriter('editing')(`A note did not close: ${formatError(error)}`, 'error'),
  )

  return {
    notes,
    making,
    fileFlush,
    store,
    getTitle: (id: string): string => (store.has(id) ? store.getTitle(id) : ''),
    close: (): void => fileFlush.close(),
  }
}
