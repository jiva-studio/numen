import type { openNotes } from '@/entities/note'
import type { Conflict, FlushHandler } from './flush'

/** What the flush needs of the notes of a window. */
type FlushedNotes = Pick<ReturnType<typeof openNotes>, 'flush' | 'getUnsavedTabs' | 'take'>

/**
 * Writes the open notes at quit. A note whose text could still not be written is a conflict: the
 * person keeps it, which writes it again, or takes the file's version, which drops the text.
 */
export function createNoteFlushHandler(notes: FlushedNotes): FlushHandler {
  return async () => {
    await notes.flush()
    return notes.getUnsavedTabs().map(
      (note): Conflict => ({
        note,
        // The run of the handlers that follows writes the text again.
        keep: () => Promise.resolve(),
        take: () => Promise.resolve(notes.take(note)),
      }),
    )
  }
}
