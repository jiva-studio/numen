/**
 * Domain types for open notes and their storage.
 */
import type { ErrorCode, NoteResult } from '../shared/core'
import type { NoteBaseline, State, waiting } from './tabState'

/** One open note as the window draws it. */
export interface OpenNote {
  readonly path: string
  readonly body: string
  readonly state: State
  readonly error: ErrorCode | null
}

/**
 * The core as a tab reads and writes through it.
 */
export interface Notes {
  read(path: string): Promise<NoteResult & { at?: string }>
  write(
    path: string,
    body: string,
    seen: NoteBaseline | null,
  ): Promise<NoteResult & { at?: string; changed?: boolean }>
}

/** What a person is told and answers with when their tab is in conflict. */
export interface ConflictWords {
  readonly says: string
  readonly keep: string
  readonly take: string
}

/** What the window hands the store of open notes, beside the vault itself. */
export interface OpenNotesOptions {
  /** How long the typing settles for, and how long a note may go unwritten. */
  limits?: typeof waiting
  /** What hears that a note on screen was replaced by what its file holds. */
  replaced?(path: string): void
  /** When it is now. */
  now?(): number
}
