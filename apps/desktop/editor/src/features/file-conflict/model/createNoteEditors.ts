/**
 * Editing, conflict handling, and flushing for open notes.
 */
import { formatError, type Caught } from '@/shared/api/errors'
import { runInBackground } from '@numen/ui'
import { useOpenNotes, flattenOpenNotes, createNoteWriter } from '@/entities/note'
import { reportFlush, watchQuit } from '@/entities/window'
import type { WindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/model/notices/messages'
import { raiseConflicts } from './conflicts'
import { createFileFlush } from './flush'
import { createNoteFlushHandler } from './noteFlush'

export interface NoteEditorsDeps {
  readonly log: MessageLog
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
}

/** What the store of the notes of the window reads and releases of the notes it opens. */
type OpenNotes = Pick<
  ReturnType<typeof flattenOpenNotes>,
  'has' | 'getPath' | 'stale' | 'flushPending' | 'close' | 'getOpenIds'
>

function createNoteStore(
  notes: OpenNotes,
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
  const notes = flattenOpenNotes(useOpenNotes({}))
  const making = createNoteWriter(log.getWriter('made'))
  const fileFlush = createFileFlush({ watchQuit, reportFlush })
  fileFlush.addHandler(createNoteFlushHandler(notes))
  raiseConflicts(notes, fileFlush)

  const store = createNoteStore(notes, windowTabs, (error) =>
    log.getWriter('editing')(`A note did not close: ${formatError(error)}`, 'error'),
  )

  return {
    notes,
    making,
    fileFlush,
    store,
    getTitle: (id: string): string => (store.has(id) ? store.getTitle(id) : id),
    close: (): void => fileFlush.close(),
  }
}
