import type { Result } from '@numen/plugin-sdk'
import type { ErrorCode } from '@/shared/errors'
import type { TextRange } from '@/shared/textRange'
import type { MoveResult, NoteType } from '@/shared/model/fileTypes'

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

/** The position of a note linked to the note in focus. */
export type NoteRelation = 'parent' | 'child' | 'jump' | 'sibling'

/** The note a neighbourhood is drawn around. */
export interface Focus {
  /** The note's current path in the vault. Empty when the vault does not contain the note. */
  readonly path: string
  readonly title: string
}

/** One note linked to the note in focus, with the label of the link. */
export interface Neighbour {
  readonly path: string
  readonly title: string
  /** Which of four it is. */
  readonly type: NoteType
  readonly relation: NoteRelation
  /** The label written on the link; empty when none is written. */
  readonly label: string
  /** What the other note wrote on its link back, when isMutual is true. */
  readonly reverseLabel?: string
  /** The parent a sibling shares with the note in focus. */
  readonly through: string
  /** Whether both notes named the relationship. */
  readonly isMutual: boolean
}

/**
 * A neighbourhood of a note, as the vault answers one: the note in focus, and
 * the notes joined to it. A note the window has no relation for is not one of them.
 */
export interface Neighbourhood {
  readonly focus: Focus
  /** Which of four the note in focus is. */
  readonly focusType: NoteType
  readonly related: readonly Neighbour[]
}

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

/** A note to make: what it is called, where it goes, and what it arrives joined to. */
export interface NewNote {
  title: string
  /** Where in the vault it goes, relative to the root. Empty is the root. */
  folder: string
  links: readonly Link[]
  /**
   * URL the note points to. When set, it becomes a link, and the vault rejects
   * an address that cannot be fetched.
   */
  url?: string
}

/**
 * The role of a link. The list is closed: navigation and
 * rendering read it, so an unlisted role has no behaviour.
 */
export type Role = 'parent' | 'child' | 'jump' | 'ref' | 'attachment'

/**
 * One relationship as declared by the note it is written in: the target note
 * by path, and the role.
 */
export interface Link {
  to: string
  role: Role
  /** The user-defined name of this relationship, when one is set. */
  label?: string
}

/** One relationship between two notes: the source note, the target note, and the label. */
export interface PairLink {
  readonly from: string
  readonly to: string
  readonly role: Role
  readonly label: string
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
