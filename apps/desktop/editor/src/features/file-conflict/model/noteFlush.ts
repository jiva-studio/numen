import type { openNotes } from '@/entities/note'
import type { Conflict, FlushHandler } from './flush'

/** What the flush needs of the notes of a window. */
type FlushedNotes = Pick<ReturnType<typeof openNotes>, 'flush' | 'getUnsavedTabs' | 'keep' | 'take'>

/**
 * Writes the open notes at quit. A note whose text could still not be written is a conflict: the
 * person keeps it, which writes it again (as the file again, when the file was deleted), or takes
 * the file's version, which drops the text. A deleted file's version is the text it last had.
 */
export function createNoteFlushHandler(notes: FlushedNotes): FlushHandler {
  return async () => {
    await notes.flush()
    return notes.getUnsavedTabs().map((note): Conflict => ({
      note,
      // A note stopped by an error is written again by the run of the handlers that follows. A note
      // whose file was deleted is written as the file again here.
      keep: () => Promise.resolve(notes.keep(note)),
      take: () => Promise.resolve(notes.take(note)),
    }))
  }
}
