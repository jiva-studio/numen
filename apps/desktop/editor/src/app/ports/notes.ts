/** What the window asks of the vault about notes. */
import type {
  Link,
  Neighbourhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteResult,
  PairLink,
  Role,
  WriteResult,
  RemoveResult,
  RenameResult,
} from '@/entities/note'
import type { ErrorCode } from '@/shared/errors'
import type { CreateResult } from '@/entities/file'

export interface NotePort {
  neighbourhood(path: string): Promise<Neighbourhood>
  /**
   * What each of the notes asked about is divided into, by the path it was
   * asked about. A note with no headings in it is absent.
   */
  headings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  /**
   * The address of each wikilink in a text, read by the core, and the note it
   * reaches: empty where it reaches none, or one in another vault. A name
   * resolves by a path relative to the note it is written in, which is `from`.
   */
  resolveLinks(from: string, text: string): Promise<ReadonlyMap<string, string>>
  /** A change being made to a note's prose, reported while it is being made. */
  watchEdits(signal: AbortSignal): AsyncIterable<NoteEdit>
  /** The prose of a note, below its frontmatter, and the file it came out of. */
  read(path: string): Promise<NoteResult>
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
    observed: { prose: string; fingerprint: string } | null,
  ): Promise<WriteResult>
  /** A note made, named after the title it is given and joined as it is written. */
  create(note: NewNote): Promise<CreateResult>
  /**
   * A relationship written into one note. The note at the other end is left
   * alone: a link is one end's account of a relationship.
   */
  join(path: string, link: Link): Promise<ErrorCode | null>
  /**
   * A relationship taken out of one note.
   */
  removeLink(path: string, to: string, role?: Role): Promise<ErrorCode | null>
  /**
   * The links between two notes made exactly the ones given: each note holds one
   * link to the other, or none. The answer says why nothing was replaced.
   */
  replaceLinks(first: string, second: string, links: readonly PairLink[]): Promise<ErrorCode | null>
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
  remove(path: string, isPermanent?: boolean): Promise<RemoveResult>
}
