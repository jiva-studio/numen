/**
 * Domain types for open notes and their storage.
 */
import type { Result } from '@numen/plugin-sdk'
import type { NoteBaseline, NoteErrorCode, State, DEFAULT_WRITE_LIMITS } from './tabState'

import type { NoteContents } from '../../lib/note'
export type { NoteContents }

/**
 * What a read came back with: the contents, or the code of why there are none.
 * A file that is not there is `missing`.
 */
export type NoteResult = Result<NoteContents, string>

/**
 * What a write came back with. A file that changed under the writer is
 * `changed`, and nothing was written to it.
 */
export type WriteResult = Result<NoteContents, string>

/** One open note as the window draws it. */
export interface OpenNote {
  readonly path: string
  readonly body: string
  readonly state: State
  readonly error: NoteErrorCode | null
}

/**
 * The core as a tab reads and writes through it.
 */
export interface Notes {
  read(path: string): Promise<NoteResult>
  write(path: string, body: string, observed: NoteBaseline | null): Promise<WriteResult>
}

export interface ConflictWords {
  readonly says: string
  readonly keep: string
  readonly take: string
}

/** What the window hands the store of open notes, beside the vault itself. */
export interface OpenNotesOptions {
  /** How long the typing settles for, and how long a note may go unwritten. */
  limits?: typeof DEFAULT_WRITE_LIMITS
  /** What hears that a note on screen was replaced by what its file holds. */
  onReplaced?(path: string): void
  /** When it is now. */
  now?(): number
}
