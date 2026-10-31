/** The vault operations on notes available to the window. */
import type {
  Neighbourhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteResult,
  WriteResult,
  RemoveResult,
  RenameResult,
} from '@/entities/note'
import type { TextLink, VaultRelations } from '@numen/plugin-sdk'
import type { CreateResult } from '@/entities/file'

export interface NotePort {
  neighbourhood(path: string): Promise<Neighbourhood>
  /**
   * The headings of each note, keyed by path. A note with no headings is absent.
   */
  headings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  /**
   * The wikilinks of a text, parsed by the core, with the note each resolves to:
   * empty when it resolves to none, or to a note in another vault. A name resolves
   * by a path relative to the note it is written in, which is `from`. `vault` is
   * the id of the vault the text is written in; empty means the vault the window
   * shows.
   */
  resolveLinks(from: string, text: string, vault?: string): Promise<readonly TextLink[]>
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
    observed: { prose: string; stat: string } | null,
  ): Promise<WriteResult>
  /** A note made, named after the title it is given and joined as it is written. */
  create(note: NewNote): Promise<CreateResult>
  /** The relationships between notes: read, written and created through the core. */
  readonly relations: VaultRelations
  /**
   * A note given a different name. Whichever of the title and the filename
   * names it is brought into line, and the file follows where a title and a
   * filename are kept as one name.
   */
  rename(path: string, title: string): Promise<RenameResult>
  /**
   * A file or folder moved out of the vault into the trash, from which it can be
   * restored. A permanent removal deletes the file from disk and cannot be undone;
   * it is available for notes only.
   */
  remove(path: string, isPermanent?: boolean): Promise<RemoveResult>
}
