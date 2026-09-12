/**
 * The files of the vault: what stands at a path, what a listing of one folder
 * reports, and what moving one comes back with.
 */
import type { MoveResult, NoteType } from '@/entities/note'
import type { ErrorCode } from '@/shared/errors'

/**
 * What the vault holds at a path. A file it holds no source for — a picture,
 * an archive — is none of the three.
 */
export type Source = 'note' | 'book' | 'recording' | 'url' | 'other'

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
export interface FileEntry {
  /** What the vault calls it, relative to the root, with forward slashes. */
  readonly path: string
  /** The last segment of the path, which is what the row shows. */
  readonly name: string
  readonly folder: boolean
  readonly isFolder?: boolean
  readonly kind: Source
  readonly type: NoteType
}
export type Entry = FileEntry

/** What moving a file or a folder came back with. */
export interface FileMoveResult {
  /** What the file did. Null when it stayed where it was. */
  moved: MoveResult | null
  error?: ErrorCode | null
}
export type Movement = FileMoveResult
