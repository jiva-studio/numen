/**
 * What a file of the vault is put in front of the person with.
 *
 * Every road to a file comes through here, holding a path and no choice: what
 * the vault says it holds there decides between the editors, the reader and the
 * player. A kind of tab hands over the way it opens a file and keeps none.
 */
import type { PlexDestination } from '@numen/ui'
import type { BookFormat, DocumentFormat, FileKind, NoteType } from '@/entities/file/@x/tab'
import type { Span } from '@/shared/span'

/**
 * What a file the window opens is opened as: which kind of note it is, or the
 * preset a fourth kind of note holds.
 */
export type EditorKind = NoteType | 'preset' | 'url'

/**
 * A file put in front of the person in one editor. A line is somewhere inside
 * the file, and what standing there comes to is the editor's own.
 */
export type FileOpener = (
  path: string,
  title: string,
  where: PlexDestination,
  line?: number,
) => void

/**
 * A source put in front of the person in the reader or in the player. The
 * spans are of the source's own text, and the person is taken to the first
 * of them.
 */
export type SourceReader = (path: string, spans: readonly Span[]) => void

/** What the window asks the vault about the file it is opening. */
export interface FileOpenerDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
}

/** A source a reader is asked for, by kind and, where one format is read its own way, by format. */
export interface ReaderKey {
  readonly kind: FileKind['kind']
  readonly format?: BookFormat | DocumentFormat
}

/** What a file the vault could not be asked about at all is opened as. */
const ORDINARY: FileKind = { kind: 'note', type: 'note' }

export function fileOpeners(vault: FileOpenerDeps) {
  /** The editor each kind of note opens in, as its kind handed it over. */
  const editors = new Map<EditorKind, FileOpener>()

  /**
   * The reader each source opens in, under the key the kind registered: the
   * kind, and the format where the kind reads one of them its own way.
   */
  const readers = new Map<string, SourceReader>()

  /** A kind of tab hands over the way it puts a file in front of the person. */
  const registerEditor = (type: EditorKind, open: FileOpener) => {
    editors.set(type, open)
  }

  /** The key a reader is held under: the kind, and the format where one claims it. */
  const keyOf = (kind: FileKind['kind'], format?: BookFormat | DocumentFormat): string =>
    format ? `${kind}/${format}` : kind

  /**
   * A kind of tab hands over the reader its sources open in. Where one format
   * of a source is read its own way, the kind that reads it that way says so.
   */
  const registerReader = (key: ReaderKey, read: SourceReader) => {
    readers.set(keyOf(key.kind, key.format), read)
  }

  /**
   * What stands at a path. Nothing stands where the vault answers nothing; a
   * vault that cannot be asked at all leaves it the ordinary note it reads as.
   */
  const fileKindAt = async (path: string): Promise<FileKind | null> => {
    try {
      return (await vault.fileKinds([path])).get(path) ?? null
    } catch {
      // A vault that cannot be asked says nothing about the file, and the tab
      // that opens is the one every file opens in.
      return ORDINARY
    }
  }

  /**
   * The reader a source opens in. One format of a source may be read its own
   * way, and the source's reader is the one where none claims the format: a
   * book that reflows is turned a spread at a time, and every other book is
   * the row of pages it is drawn as.
   */
  const readerOf = (kind: FileKind): SourceReader | undefined =>
    readers.get(keyOf(kind.kind, kind.format)) ?? readers.get(keyOf(kind.kind))

  /**
   * A file just made here, put in front of the person as what it was made as.
   * The vault is not asked what stands there.
   */
  const openNewFile = (
    path: string,
    title: string,
    type: EditorKind,
    where: PlexDestination = 'here',
    line?: number,
  ): void => {
    editors.get(type)?.(path, title, where, line)
  }

  /**
   * A file put in front of the person in the editor made for what it is, in the
   * reader where a document stands there, or in the player where a recording
   * does. A path holding no source at all opens nothing.
   */
  const openFile = async (
    path: string,
    title = '',
    where: PlexDestination = 'here',
    line?: number,
  ): Promise<void> => {
    const kind = await fileKindAt(path)
    if (!kind) return
    if (kind.kind === 'note') return void openNewFile(path, title, kind.type, where, line)
    readerOf(kind)?.(path, [])
  }

  /**
   * A source put in front of the person at spans of its own text: a book in
   * the reader and a recording in the player, at the first of them. A span
   * of a note's bytes names no line for the keyboard to stand on, so a note
   * opens whole.
   */
  const openFileAt = async (path: string, spans: readonly Span[]): Promise<void> => {
    const kind = await fileKindAt(path)
    if (!kind) return
    if (kind.kind === 'note') return void openNewFile(path, '', kind.type)
    readerOf(kind)?.(path, spans)
  }

  return { registerEditor, registerReader, openFile, openFileAt, openNewFile }
}

/** What the window puts files in front of the person with. */
export type FileOpeners = ReturnType<typeof fileOpeners>
