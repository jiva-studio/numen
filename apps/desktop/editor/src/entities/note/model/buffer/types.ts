/**
 * Domain types for open notes and their storage.
 */
import type { Result } from '@numen/result'
import type { NoteErrorCode, State, DEFAULT_WRITE_LIMITS } from './tabState'

import type { NoteContents } from '../../lib/note'

/**
 * What a read returned: the contents, or the code of why there are none.
 * A file that is not there is `notFound`.
 */
export type NoteResult = Result<NoteContents, string>

/**
 * What a write returned. A file that changed under the writer is
 * `changed`, and nothing was written to it.
 */
export type WriteResult = Result<NoteContents, string>

/** One open note as the window draws it. */
export interface OpenNote {
  readonly path: string
  readonly body: string
  readonly state: State
  readonly error: NoteErrorCode | null
  readonly isReadOnly: boolean
}

export interface ConflictWords {
  readonly says: string
  readonly keep: string
  readonly take: string
}

/** What the window hands the store of open notes, beside the vault itself. */
export interface OpenNotesOptions {
  limits?: typeof DEFAULT_WRITE_LIMITS
  /** Called when a note on screen is replaced by the content of its file. */
  onReplaced?(this: void, path: string): void
  /** Whether the vault the notes are in cannot be written. */
  isReadOnly?(this: void): boolean
  /** When it is now. */
  now?(this: void): number
}
