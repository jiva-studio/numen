/**
 * The words the vault speaks about notes, and what the window asks of it over
 * them.
 */
import type { Result } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'
import type { Span } from '@/shared/span'
import type { MoveResult, NoteType } from '@/entities/file/@x/note'

/**
 * One report of a change being made to the prose of a note, while it is being
 * made: which change it belongs to, where it lands, and what goes in.
 */
export interface NoteEdit {
  readonly change: string
  readonly path: string
  readonly span: Span
  readonly text: string
  readonly isComplete: boolean
}

/** Where a note joined to the note in focus sits around it. */
export type Seat = 'parent' | 'child' | 'jump' | 'sibling'

/** The note a neighbourhood is drawn around. */
export interface Focus {
  /** Where it stands in the vault. Empty for a note the vault no longer holds. */
  readonly path: string
  readonly title: string
}

/** One note joined to the note in focus, and what the line between them says. */
export interface Neighbour {
  readonly path: string
  readonly title: string
  /** Which of four it is. */
  readonly type: NoteType
  readonly seat: Seat
  /** What the person wrote on the link, and nothing where they wrote nothing. */
  readonly label: string
  /** The parent a sibling shares with the note in focus. */
  readonly through: string
  /** Whether both notes named the relationship. */
  readonly isMutual: boolean
}

/**
 * A neighbourhood of a note, as the vault answers one: the note in focus, and
 * the notes joined to it. A note the window has no seat for is not one of them.
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
  /** How deep it sits, from one for the shallowest a note can carry. */
  readonly level: number
  /** The line it stands on, counted from the first line of the prose. */
  readonly line: number
}

/** The prose a read gave back, and what the vault says came with it. */
export interface NoteContents {
  readonly body: string
  /** Where a link note points, and nothing on every other note. */
  readonly link?: LinkAddress
  /** The file the prose came out of, as the next write presents it again. */
  readonly at?: string
}

/**
 * Why a note was not read or not written. Changed is the file holding prose
 * nobody here has seen, which stops a write.
 */
export type NoteFailure = ErrorCode | 'changed'

/**
 * What a read came back with. The words for a failure belong to whatever shows
 * it.
 */
export type NoteResult = Result<NoteContents, ErrorCode>

/** What a write came back with. A file that changed is one nothing was written to. */
export type WriteResult = Result<NoteContents, NoteFailure>

/**
 * Where a link note points: the web address (URL) and embed player URL.
 */
export interface LinkAddress {
  readonly url: string
  readonly embed: string
}

/** A note to make: what it is called, where it goes, and what it arrives joined to. */
export interface NewNote {
  title: string
  /** Where in the vault it goes, relative to the root. Empty is the root. */
  folder: string
  links: readonly Link[]
  /**
   * Where the note points. Given, it is made a link, and the vault refuses an
   * address nothing can be fetched from.
   */
  url?: string
}

/**
 * What kind of relationship a link is. The list is closed: navigation and
 * drawing read it, so a role nobody decided on has no behaviour.
 */
export type Role = 'parent' | 'child' | 'jump' | 'ref' | 'attachment'

/**
 * One relationship as the note it is written in declares it: the note at the
 * other end, by the path it is filed under, and what kind of relationship it
 * is.
 */
export interface Link {
  to: string
  role: Role
  /** What the person calls this relationship, when they call it anything. */
  label?: string
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

/** A note taken out of the vault, and what its going left behind. */
export interface RemovedNote {
  /** Where the note sits in the trash. Empty when it was destroyed. */
  readonly trashed: string
  /** The notes whose links pointed at it and now reach nothing. */
  readonly dangling: readonly string[]
}

/** What removing a note came back with. */
export type RemoveResult = Result<RemovedNote, ErrorCode>
