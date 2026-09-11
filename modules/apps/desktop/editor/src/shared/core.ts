/**
 * Everything the window asks of the vault, and the words the vault speaks.
 *
 * Nothing here is about drawing: a kind translates these into what it holds,
 * and this is what every one of them starts from.
 */
import type {
  CreateResult,
  ErrorCode,
  Link,
  PathRename,
  Neighbourhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteResult,
  RemoveResult,
  RenameResult,
} from './note'
import type { Entry, FileKind, Movement } from './file'
import type { Task } from './notices/task'
import type { Configuration, SettingEdit } from './settings/configuration'
import type { HangingSettings } from './settings/hanging'
import type { ReviewSettings } from './settings/review'
import type { Attention } from './tabs/tab'

export interface Core {
  neighbourhood(path: string): Promise<Neighbourhood>
  /**
   * What each of the notes asked about is divided into, by the path it was
   * asked about. A note with no headings in it is absent.
   */
  headings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  /**
   * What stands at each of those paths, by the path it was asked about. The
   * kind comes off the vault itself, so a path nothing has scanned is answered
   * with what stands there; a path with nothing at it is absent.
   */
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  /**
   * Where each of those addresses lands, by the address it was asked about. A
   * name resolves by a path relative to the note it is written in, which is
   * `from`; an address that reaches nothing is absent.
   */
  resolve(from: string, written: readonly string[]): Promise<ReadonlyMap<string, string>>
  opening(): Promise<{ path: string } | null>
  state(): Promise<{
    /** The identity the folder carries, which is how the vault is asked for again. */
    id: string
    /** What the person calls the vault. */
    name: string
    /** The folder the vault sits in, absolute on this machine. */
    path: string
    /** How far reading the vault has got. */
    scan: {
      isReady: boolean
      failureReason: string
      unwatchedPath: string
    }
    /** How far searching it by meaning has got. */
    coverage: {
      chunkCount: bigint
      embeddedCount: bigint
      isEmbedding: boolean
    }
  }>
  /**
   * Why an agent cannot be reached, and nothing where one can. Which agent
   * answers is the installation's, so it is asked of the agent and not of the
   * vault the window is showing.
   */
  agentUnreachable(): Promise<string>
  changes(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    shouldReload: boolean
    renamed: readonly PathRename[]
  }>
  /** A change being made to a note's prose, reported while it is being made. */
  editing(signal: AbortSignal): AsyncIterable<NoteEdit>
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
   * source, and the span of its own text meant, counted in bytes. A length
   * of zero names the source and no place inside it.
   */
  focus(signal: AbortSignal): AsyncIterable<{
    path: string
    spans: readonly { from: number; to: number }[]
  }>
  /**
   * What the person has open, said again whenever any of it changes. It is the
   * other direction to `focus`: a place is put in front of the person there,
   * and here the window says what is in front of them now.
   */
  attending(open: Attention): Promise<void>
  /** The prose of a note, below its frontmatter, and the file it came out of. */
  read(path: string): Promise<NoteResult & { at?: string }>
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
  ): Promise<NoteResult & { at?: string; changed?: boolean }>
  /** A note made, named after the title it is given and joined as it is written. */
  create(note: NewNote): Promise<CreateResult>
  /**
   * A relationship written into one note. The note at the other end is left
   * alone: a link is one end's account of a relationship.
   */
  join(path: string, link: Link): Promise<ErrorCode | null>
  /**
   * A note given a different name. Whichever of the title and the filename
   * names it is brought into line, and the file follows where a title and a
   * filename are kept as one name.
   */
  rename(path: string, title: string): Promise<RenameResult>
  /**
   * A file or a folder taken out of the vault, into the trash it can be brought
   * back from. Destroying takes the file off the disk and brings nothing back,
   * and is asked of a note only.
   */
  remove(path: string, destroy?: boolean): Promise<RemoveResult>
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
  choosesSyncing(kept: boolean): Promise<string | null>
  /**
   * Whether a node in the plex hangs the parts of its note under the box, and
   * how many of them stand there at once, as the settings hold them.
   */
  hanging(): Promise<HangingSettings>
  /**
   * Those settings written into the settings file. What could not be written,
   * and nothing where it was. A count left out stands as it is.
   */
  choosesHanging(hangs: boolean, parts?: number): Promise<string | null>
  /**
   * The hour a day of review begins at, on the clock on the wall, written as
   * `04:00`, and how late in the day the vault takes one. An hour past that is
   * refused.
   */
  reviewing(): Promise<ReviewSettings>
  /**
   * That hour written into the settings file. What could not be written, and
   * nothing where it was.
   */
  choosesReviewing(starts: string): Promise<string | null>
  /** Every setting as it stands, and the models the settings offer. */
  settings(): Promise<Configuration>
  /**
   * Settings written into the settings file, together or not at all. A value
   * the settings could not be read out of again is refused, and what the file
   * holds is unchanged.
   */
  choosesSetting(written: readonly SettingEdit[]): Promise<void>
  /** The settings file as its person wrote it, and where it stands. */
  settingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * The settings file replaced whole, with the bytes as they were typed. A file
   * the settings could not be read out of is refused, and what the file holds
   * is unchanged.
   *
   * Seen is the file as it was last read, and a file standing at anything else
   * is answered `changed` with nothing written. Nothing seen writes over
   * whatever the file holds.
   */
  writesSettingsFile(
    written: string,
    seen: string | null,
  ): Promise<{ readonly changed: boolean }>
  /** An empty folder. The folders above it are made with it. */
  createFolder(path: string): Promise<ErrorCode | null>
  /**
   * The file a web address is kept in, named by the address until a fetch says
   * what is there.
   */
  createUrl(url: string, folder: string): Promise<CreateResult>
  /**
   * The window going, for as long as the client listens. The stream opens with
   * the token this client answers under.
   */
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Everything this client owed has been written. */
  flushed(token: string, owed?: 'written' | 'asking'): Promise<void>
}

export * from './note'
export type * from './file'
export * from './artifacts'
export type * from './notices/task'
export type * from './settings/configuration'
export type { HangingSettings } from './settings/hanging'
export type { ReviewSettings } from './settings/review'
export type * from './tabs/tab'
export type * from './vaults'
export * from './result'
