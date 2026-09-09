/**
 * The files of the vault: what stands at a path, what a listing of one folder
 * reports, and what moving one comes back with.
 */
import type { MoveResult, NoteType, RefusalReason } from './note'

/**
 * What the vault holds at a path. A file it holds no source for — a picture,
 * an archive — is none of the three.
 */
export type Source = 'note' | 'book' | 'recording' | 'url' | 'other'

/**
 * Which sort of book stands at a path: one drawn as pictures a page at a time,
 * or one that reflows to the room it is read in.
 */
export type BookFormat = 'pdf' | 'epub'

/** What stands at a path: which source it is, and which kind of note it is. */
export interface FileKind {
  readonly kind: Source
  readonly type: NoteType
  /** Which sort of book it is. It says nothing about a path holding no book. */
  readonly format?: BookFormat
}

/** One file or folder, as a listing of the folder it sits in reports it. */
export interface Entry {
  /** What the vault calls it, relative to the root, with forward slashes. */
  readonly path: string
  /** The last segment of the path, which is what the row shows. */
  readonly name: string
  readonly folder: boolean
  readonly kind: Source
  readonly type: NoteType
}

/** What moving a file or a folder came back with. */
export interface Movement {
  /** What the file did. Null when it stayed where it was. */
  moved: MoveResult | null
  refusal: RefusalReason | null
}
