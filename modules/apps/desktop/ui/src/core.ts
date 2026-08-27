/**
 * Everything the window asks of the vault, and the words the vault speaks.
 *
 * Nothing here is about drawing: a kind translates these into what it holds,
 * and this is what every one of them starts from.
 */
import type { Counting } from '@numen/ui'
import type { NeighbourhoodResponse } from '@numen/protocol'

/** A note that is no longer where it was, and where it now is. */
export interface Went {
  readonly from: string
  readonly to: string
}

/** Where a note went, and nothing where none of these moved it. */
export const wentTo = (renamed: readonly Went[], path: string): string =>
  renamed.find((one) => one.from === path)?.to ?? ''

/** A stretch of a source's own text, counted in bytes. */
export interface Run {
  readonly start: number
  readonly length: number
}

/**
 * One report of a change being made to the prose of a note, while it is being
 * made: which change it belongs to, where it lands, and what goes in.
 */
export interface Said {
  readonly change: string
  readonly path: string
  readonly from: number
  readonly to: number
  readonly text: string
  readonly done: boolean
}

/** A neighbourhood of a note, as the vault answers one. */
export type Neighbourhood = NeighbourhoodResponse

/** One heading inside a note, which is one of the parts the note divides into. */
export interface Heading {
  readonly text: string
  /** How deep it sits, from one for the shallowest a note can carry. */
  readonly level: number
  /** The line it stands on, counted from the first line of the prose. */
  readonly line: number
}

/**
 * What the vault holds at a path. A file it holds no source for — a picture,
 * an archive — is neither of the two.
 */
export type Source = 'note' | 'book' | 'other'

/** One file or folder, as a listing of the folder it sits in reports it. */
export interface Entry {
  /** What the vault calls it, relative to the root, with forward slashes. */
  readonly path: string
  /** The last segment of the path. */
  readonly name: string
  readonly folder: boolean
  readonly kind: Source
}

/** What moving a file or a folder came back with. */
export interface Movement {
  /** What the file did. Null when it stayed where it was. */
  moved: Moved | null
  refusal: Refused | null
}

/**
 * One piece of work the application is doing behind the window.
 *
 * Every kind of work is one of these, which is what keeps the window from
 * growing a branch per kind: it draws the list it is given.
 */
export interface Task {
  /** What the work is called, so that the same work reported again replaces it. */
  readonly id: string
  /** The work, in the words to show, and what it is on. */
  readonly doing: string
  readonly about: string
  /** How far it has got, where there is a total to count against. */
  readonly done: number
  readonly total: number
  /** What that count counts. */
  readonly counting: Counting
  /** Why it stopped, when it stopped badly. */
  readonly failed: string
  /** Whether a person asked for this and is waiting to be told it began. */
  readonly asked: boolean
}

export interface Core {
  neighbourhood(path: string): Promise<Neighbourhood>
  /**
   * What each of the notes asked about is divided into, by the path it was
   * asked about. A note with no headings in it is absent.
   */
  headings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly Heading[]>>
  opening(): Promise<{ path: string } | null>
  state(): Promise<{
    name: string
    /** The folder the vault sits in, absolute on this machine. */
    path: string
    ready: boolean
    failed: string
    unwatched: string
    unreachable: string
    /** Spans of text the index holds, and how many of them carry a vector. */
    chunks: bigint
    embedded: bigint
    /** Whether anything is going to turn the chunks into vectors. */
    embedding: boolean
  }>
  changes(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    reload: boolean
    renamed: readonly Went[]
  }>
  /** A change being made to a note's prose, reported while it is being made. */
  editing(signal: AbortSignal): AsyncIterable<Said>
  /**
   * Everything the application is doing behind the window, for as long as the
   * window listens.
   *
   * The whole list arrives whenever any of it changes, and the first arrives at
   * once. It is a stream because work can begin without the window asking for
   * it: an agent is told to read a document, and this is where the person
   * watching sees it happen.
   */
  tasks(signal: AbortSignal): AsyncIterable<readonly Task[]>
  /**
   * The places something else asked to be put in front of the person: a
   * source, and the stretch of its own text meant, counted in bytes. A length
   * of zero names the source and no place inside it.
   */
  focus(signal: AbortSignal): AsyncIterable<{
    path: string
    start?: number
    length?: number
    also?: readonly { start?: number; length?: number }[]
  }>
  /** The prose of a note, below its frontmatter, and the file it came out of. */
  read(path: string): Promise<Answered & { at?: string }>
  /**
   * Prose into a note, keeping the frontmatter the file has when it lands.
   *
   * Seen is what a read gave this caller. Prose on disk that the caller never
   * saw comes back as changed, and nothing is written. Nothing seen writes
   * over whatever is there.
   */
  write(
    path: string,
    body: string,
    seen: { prose: string; at: string } | null,
  ): Promise<Answered & { at?: string; changed?: boolean }>
  /** A note made, named after the title it is given and joined as it is written. */
  create(note: NewNote): Promise<Made>
  /**
   * A relationship written into one note. The note at the other end is left
   * alone: a link is one end's account of a relationship.
   */
  join(path: string, link: NewLink): Promise<Refused | null>
  /**
   * A note given a different name. Whichever of the title and the filename
   * names it is brought into line, and the file follows where a title and a
   * filename are kept as one name.
   */
  rename(path: string, title: string): Promise<Renamed>
  /**
   * A file or a folder taken out of the vault, into the trash it can be brought
   * back from. Destroying takes the file off the disk and brings nothing back,
   * and is asked of a note only.
   */
  remove(path: string, destroy?: boolean): Promise<Removed>
  /**
   * What one folder of the vault holds, in the order to draw it: folders first
   * and then files, each group by name with case ignored. The root is the empty
   * path, and hidden files are in none of the answers.
   */
  list(folder: string): Promise<readonly Entry[]>
  /**
   * A file or a folder filed somewhere else. The last segment of `to` is what
   * it is called from now on, so a name changed within one folder is a move.
   */
  move(from: string, to: string): Promise<Movement>
  /**
   * Whether renaming either a note's title or the name of its file brings the
   * other into line, as the settings hold it.
   */
  syncing(): Promise<boolean>
  /**
   * That setting written into the settings file. What could not be written, and
   * nothing where it was: the rename after this reads what was written.
   */
  choosesSyncing(kept: boolean): Promise<Refused | null>
  /**
   * Whether a node in the plex hangs the parts of its note under the box, as
   * the settings hold it.
   */
  hanging(): Promise<boolean>
  /**
   * That setting written into the settings file. What could not be written, and
   * nothing where it was.
   */
  choosesHanging(hangs: boolean): Promise<Refused | null>
  /** An empty folder. The folders above it are made with it. */
  makeFolder(path: string): Promise<Refused | null>
  /**
   * The window going, for as long as the client listens. The stream opens with
   * the token this client answers under.
   */
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Everything this client owed has been written. */
  flushed(token: string, owed?: 'written' | 'asking'): Promise<void>
}

/**
 * What a read or a write came back with. A refusal carries no body, and the
 * words for one belong to whatever shows it.
 */
export interface Answered {
  body: string
  refusal: Refused | null
}

export type Refused =
  | 'missing'
  | 'notANote'
  | 'notText'
  | 'tooLarge'
  | 'bodyRefused'
  | 'unreadable'
  | 'occupied'
  | 'unnameable'

/** A note to make: what it is called, where it goes, and what it arrives joined to. */
export interface NewNote {
  title: string
  /** Where in the vault it goes, relative to the root. Empty is the root. */
  folder: string
  links: readonly NewLink[]
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
export interface NewLink {
  to: string
  role: Role
  /** What the person calls this relationship, when they call it anything. */
  label?: string
}

/** What making a note came back with. */
export interface Made {
  /** Where the note is filed. Empty when nothing was made. */
  path: string
  refusal: Refused | null
}

/** What renaming a note came back with. */
export interface Renamed {
  /**
   * Where the note is filed. The note is brought into line before the file is,
   * so a refused move comes back with the path the note still has.
   */
  path: string
  title: string
  /** Whether the rename wrote the title into the frontmatter of the note. */
  frontmatter: boolean
  /** What the file did. Null when it stayed where it was. */
  moved: Moved | null
  refusal: Refused | null
  /** The note holds prose nobody here has seen, and nothing was written. */
  changed: boolean
}

/**
 * A file under a different name, and what that did to the links written by the
 * name it had.
 */
export interface Moved {
  readonly from: string
  readonly to: string
  /** The notes whose link stopped resolving and was written again, by name. */
  readonly repaired: readonly string[]
}

/** What removing a note came back with. */
export interface Removed {
  /** Where the note sits in the trash. Empty when it was destroyed. */
  trashed: string
  /** The notes whose links pointed at it and now reach nothing. */
  dangling: readonly string[]
  refusal: Refused | null
}

/** One vault the installation holds, as the list has it. */
export interface Known {
  /** The identity the folder carries, and how the vault is asked for again. */
  readonly id: string
  /** What the person calls the collection. */
  readonly name: string
  /** The folder, absolute on this machine. */
  readonly path: string
  /** Whether nothing is at the path. The vault stays on the list. */
  readonly missing: boolean
}

/** Every vault the installation holds, and the one this window is showing. */
export interface Listed {
  readonly vaults: readonly Known[]
  /** The identity of the vault in front of the person. */
  readonly showing: string
}

/** What adding a vault came back with, and what renaming one comes back with. */
export interface Added {
  /** The vault as the list has it now. Null where the list is as it was. */
  vault: Known | null
  refusal: VaultRefused | null
}

/** Why the list is as it was, or why the window is showing what it was showing. */
export type VaultRefused =
  | 'unreadable'
  | 'copy'
  | 'overlaps'
  | 'nameTaken'
  | 'lastVault'
  | 'showing'
  | 'unknown'
  | 'noTrash'
  | 'asking'

/** The vaults an installation holds, and what changes them. */
export interface Vaults {
  /** Every vault on the list, and which of them this window is showing. */
  list(): Promise<Listed>
  /**
   * This machine's own folder picker, put in front of the person. It answers
   * with the folder they chose, and with nothing where they closed it.
   */
  choose(title: string): Promise<string>
  /**
   * A folder turned into a vault and put on the list. The folder is given an
   * identity that stays with it wherever it moves to.
   */
  add(path: string, name: string): Promise<Added>
  /** What a person calls a vault. The folder keeps the name the filesystem gives it. */
  rename(id: string, name: string): Promise<Added>
  /** A vault taken off the list. The folder stays where it is. */
  forget(id: string): Promise<VaultRefused | null>
  /** A vault taken off the list, and its folder into the trash this machine keeps. */
  erase(id: string): Promise<VaultRefused | null>
  /** Another vault shown in this window, in place of the one it was showing. */
  open(id: string): Promise<VaultRefused | null>
}
