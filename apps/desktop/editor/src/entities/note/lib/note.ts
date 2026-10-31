import type { NoteNeighbourhood, Result } from '@numen/plugin-sdk'
import type { ErrorCode } from '@/shared/errors'
import type { TextRange } from '@/shared/textRange'
import type { MoveResult } from '@/shared/model/fileTypes'

export type { NoteRelation } from '@numen/plugin-sdk'

/** The prose returned by a read, with the file identity returned alongside it. */
export interface NoteContents {
  readonly body: string
  /** The file the prose came out of, as the next write presents it again. */
  readonly at?: string
}

/**
 * One report of a change being made to the prose of a note, while it is being
 * made: which change it belongs to, where it lands, and what goes in.
 */
export interface NoteEdit {
  readonly change: string
  readonly path: string
  readonly span: TextRange
  readonly text: string
  readonly isComplete: boolean
}

/**
 * A neighbourhood of a note, as the vault answers one: the note in focus, and
 * the notes joined to it. A note the window has no relation for is not one of them.
 */
export type Neighbourhood = NoteNeighbourhood

/** One heading inside a note, which is one of the parts the note divides into. */
export interface NoteHeading {
  readonly text: string
  /** Nesting depth, starting at one for the shallowest level a note can have. */
  readonly level: number
  /** The line it relies on, counted from the first line of the prose. */
  readonly line: number
}

/**
 * Why a note was not read or not written. `changed` means the file contains
 * prose that has not been loaded, which stops a write.
 */
export type NoteFailure = ErrorCode | 'changed'

/**
 * What a read came back with. The words for a failure belong to whatever shows
 * it.
 */
export type NoteResult = Result<NoteContents, ErrorCode>

/** What a write came back with. A file that changed is one nothing was written to. */
export type WriteResult = Result<NoteContents, NoteFailure>

/** A note to make: what it is called, and where it goes. */
export interface NewNote {
  /** What it is called. Empty takes the lowest free "Untitled note N". */
  title: string
  /** Where in the vault it goes, relative to the root. Empty is the root. */
  folder: string
}

/** A note renamed, and what its file did. */
export interface RenamedNote {
  /** Where the note is filed now. */
  readonly path: string
  readonly title: string
  /** Whether the rename wrote the title into the frontmatter of the note. */
  readonly hasFrontmatter: boolean
  /** What the file did. Null when it stayed where it was. */
  readonly moved: MoveResult | null
}

/** What renaming a note came back with. */
export type RenameResult = Result<RenamedNote, NoteFailure>

export interface RemovedNote {
  readonly trashed: string
  /** The notes whose links pointed at it and now reach nothing. */
  readonly dangling: readonly string[]
}

/** What removing a note came back with. */
export type RemoveResult = Result<RemovedNote, ErrorCode>
