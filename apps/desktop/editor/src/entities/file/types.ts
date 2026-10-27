/**
 * The files of the vault: which of them a file is, what a listing of one folder
 * reports, and what making one or moving one comes back with.
 */
import type { Result } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'

/**
 * What a note is, as the `type` key of its frontmatter says: `note`, or a type a plugin declares.
 * It says nothing about a file that is not a note.
 */
export type NoteType = string

/** Whether a file is a note, and which type of note. A file that is no note opens in the reader of its content type. */
export interface FileKind {
  readonly isNote: boolean
  readonly type: NoteType
}

export interface Entry {
  /** What the vault calls it, relative to the root, with forward slashes. */
  readonly path: string
  /** The last segment of the path, which is what the row shows. */
  readonly name: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly type: NoteType
}

/** Where a note was filed, or why none was made. */
export type CreateResult = Result<{ readonly path: string }, ErrorCode>

/**
 * A file under a different name, and what that did to the links written by the
 * name it had.
 */
export interface MoveResult {
  readonly from: string
  readonly to: string
  /** The notes whose link stopped resolving and was written again, by name. */
  readonly repaired: readonly string[]
}

/**
 * What moving a file or a folder did, or why nothing was moved. A value of
 * null is a move that was asked for and changed nothing.
 */
export type Movement = Result<MoveResult | null, ErrorCode>
