/**
 * The words the vault speaks about notes, and what the window asks of it over
 * them.
 */
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

/**
 * What a read or a write came back with. An error carries no body, and the
 * words for one belong to whatever shows it.
 */
export interface NoteResult {
  body: string
  error?: ErrorCode | null
  /** Where a link note points, and nothing on every other note. */
  link?: LinkAddress
}

/**
 * What a read came back with: the note, and which bytes the file was when it
 * was read. The fingerprint is what a later write presents to say which
 * version it was typed over.
 */
export interface NoteReadResult extends NoteResult {
  fingerprint?: string
}

/**
 * What a write came back with. `isChanged` is the file having moved past the
 * fingerprint the write presented, so nothing was written and what is on disk
 * is somebody else's.
 */
export interface NoteWriteResult extends NoteReadResult {
  isChanged?: boolean
}

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

/** What renaming a note came back with. */
export interface RenameResult {
  /**
   * Where the note is filed. The note is brought into line before the file is,
   * so a refused move comes back with the path the note still has.
   */
  path: string
  title: string
  /** Whether the rename wrote the title into the frontmatter of the note. */
  hasFrontmatter: boolean
  /** What the file did. Null when it stayed where it was. */
  moved: MoveResult | null
  error?: ErrorCode | null
  /** The note holds prose nobody here has seen, and nothing was written. */
  hasChanged: boolean
}

/** What removing a note came back with. */
export interface RemoveResult {
  /** Where the note sits in the trash. Empty when it was destroyed. */
  trashed: string
  /** The notes whose links pointed at it and now reach nothing. */
  dangling: readonly string[]
  error?: ErrorCode | null
}
