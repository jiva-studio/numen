/**
 * The files of the vault: which of them a file is, what a listing of one folder
 * reports, and what making one or moving one comes back with.
 */
import type { Result } from '@numen/core-client'
import type { ErrorCode } from '@/shared/errors'

/**
 * The note type from the `type` key of its frontmatter: `note`, or a type a plugin declares.
 * It does not apply to a file that is not a note.
 */
export type NoteType = string

/**
 * Whether a file is a note, and which type of note. A note opens in the tab kind the core chooses for
 * it; a note no tab kind takes and a file that is no note open in the reader of their content type.
 */
export interface FileKind {
  readonly isNote: boolean
  readonly type: NoteType
  /** The kind of tab the note opens in; absent where no tab kind takes it. */
  readonly tabKind?: string
  /** The frontmatter of a note by its keys; absent for a file that is no note or a note without any. */
  readonly properties?: Readonly<Record<string, unknown>>
}

export interface Entry {
  readonly path: string
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
  readonly repaired: readonly string[]
}

/**
 * The result of moving a file or a folder, or the reason nothing was moved. A
 * value of null is a requested move that changed nothing.
 */
export type Movement = Result<MoveResult | null, ErrorCode>
