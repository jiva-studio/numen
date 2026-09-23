/**
 * The files of the vault: which of them a file is, what a listing of one folder
 * reports, and what making one or moving one comes back with.
 */
import type { Result } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'

/**
 * What the vault holds at a path. A file it holds no source for — a picture,
 * an archive — is none of the four.
 */
export type Source = 'note' | 'book' | 'recording' | 'url' | 'other'

/**
 * Which of four a file is, as the `type` key of its frontmatter says. It says
 * nothing about a file that is not a note.
 */
export type NoteType = 'note' | 'deck' | 'stencil' | 'preset'

/** Which sort of document or book stands at a path. */
export type DocumentFormat = 'pdf'
export type BookFormat = 'epub'

/** What stands at a path: which source it is, and which kind of note it is. */
export interface FileKind {
  readonly kind: Source
  readonly type: NoteType
  /** Which sort of book or document it is. */
  readonly format?: DocumentFormat | BookFormat
}

/** One file or folder, as a listing of the folder it sits in reports it. */
export interface Entry {
  /** What the vault calls it, relative to the root, with forward slashes. */
  readonly path: string
  /** The last segment of the path, which is what the row shows. */
  readonly name: string
  readonly isFolder: boolean
  readonly kind: Source
  readonly type: NoteType
}

/** Where a note, a deck or a stencil was filed, or why none was made. */
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
