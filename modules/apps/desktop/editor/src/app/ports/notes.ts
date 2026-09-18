/** What the window asks of the vault about notes. */
import type {
  Link,
  Neighbourhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteReadResult,
  NoteWriteResult,
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
   * Where each of those addresses lands, by the address it was asked about. A
   * name resolves by a path relative to the note it is written in, which is
   * `from`; an address that reaches nothing is absent.
   */
  resolve(from: string, written: readonly string[]): Promise<ReadonlyMap<string, string>>
  /** A change being made to a note's prose, reported while it is being made. */
  watchEdits(signal: AbortSignal): AsyncIterable<NoteEdit>
  /** The prose of a note, below its frontmatter, and the file it came out of. */
  read(path: string): Promise<NoteReadResult>
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
    seen: { prose: string; fingerprint: string } | null,
  ): Promise<NoteWriteResult>
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
  remove(path: string, isPermanent?: boolean): Promise<RemoveResult>
}
