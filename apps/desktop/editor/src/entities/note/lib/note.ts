import type { Result } from '@/shared/lib/result'
import type { NoteNeighborhood } from '@numen/plugin-loader'
import type { ErrorCode } from '@/shared/api/errors'
import type { TextRange } from '@/shared/lib/textRange'
import type { MoveResult } from '@/shared/model/fileTypes'

export type { NoteRelation } from '@numen/plugin-loader'

/** The prose returned by a read, with the file identity returned alongside it. */
export interface NoteContents {
  readonly body: string
  readonly at?: string
}

/**
 * One report of a change being made to the prose of a note, while it is being
 * made: which change it belongs to, its position, and what goes in.
 */
export interface NoteEdit {
  readonly change: string
  readonly path: string
  readonly span: TextRange
  readonly text: string
  readonly isComplete: boolean
}

/**
 * A neighborhood of a note, as the vault answers one: the note in focus, and
 * the notes joined to it. A note the window has no relation for is not one of them.
 */
export type Neighborhood = NoteNeighborhood

/** One heading inside a note, which is one of the parts the note divides into. */
export interface NoteHeading {
  readonly text: string
  readonly level: number
  readonly line: number
}

/**
 * Why a note was not read or not written. `changed` means the file contains
 * prose that has not been loaded, which stops a write.
 */
type NoteFailure = ErrorCode | 'changed'

/**
 * What a read returned. The words for a failure belong to whatever shows
 * it.
 */
export type NoteResult = Result<NoteContents, ErrorCode>

/** What a write returned. A file that changed is one nothing was written to. */
export type WriteResult = Result<NoteContents, NoteFailure>

/** A note to make: what it is called, and where it goes. */
export interface NewNote {
  title: string
  folder: string
}

/** A note renamed, and what its file did. */
interface RenamedNote {
  readonly path: string
  readonly title: string
  readonly hasFrontmatter: boolean
  readonly moved: MoveResult | null
}

/** What renaming a note returned. */
export type RenameResult = Result<RenamedNote, NoteFailure>
