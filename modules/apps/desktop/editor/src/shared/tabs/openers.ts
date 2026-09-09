/**
 * What a file of the vault is put in front of the person with.
 *
 * Every road to a file comes through here, holding a path and no choice: what
 * the vault says it holds there decides between the editors, the reader and the
 * player. A kind of tab hands over the way it opens a file and keeps none.
 */
import type { PlexShowing } from '@numen/ui'
import { troubleWords } from '@numen/wire'
import type { FileKind } from '../file'
import type { MakeResult, NoteType, RefusalReason, Span } from '../note'
import type { MessageWriter } from '../notices/messages'

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
  showing: PlexShowing,
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

/** What a file the vault could not be asked about at all is opened as. */
const ORDINARY: FileKind = { kind: 'note', type: 'note' }

export function fileOpeners(vault: FileOpenerDeps) {
  /** The editor each kind of note opens in, as its kind handed it over. */
  const editors = new Map<EditorKind, FileOpener>()

  /**
   * The reader a document opens in and the player a recording is heard in, each
   * under the kind the vault answers a path with.
   */
  const sources = new Map<FileKind['kind'], SourceReader>()

  /** A kind of tab hands over the way it puts a file in front of the person. */
  const holds = (type: EditorKind, opens: FileOpener) => {
    editors.set(type, opens)
  }

  /** The reader a book that reflows opens in, which draws no pages. */
  let turning: SourceReader | null = null

  /** The kind of tab that reads documents hands its own over. */
  const reads = (opens: SourceReader) => {
    sources.set('book', opens)
  }

  /** The kind of tab that turns a book that reflows hands its own over. */
  const turns = (opens: SourceReader) => {
    turning = opens
  }

  /** The kind of tab that plays recordings hands its own over. */
  const hears = (opens: SourceReader) => {
    sources.set('recording', opens)
  }

  /**
   * The kind of tab that opens urls hands its own over. A url is reached both
   * ways: by what the vault says stands at a path, and by having just been made
   * here.
   */
  const points = (opens: SourceReader) => {
    sources.set('url', opens)
    editors.set('url', (path) => opens(path, []))
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
   * The reader a source opens in. A book that reflows is turned a spread at a
   * time, and every other book is the row of pages it is drawn as.
   */
  const readerOf = (stands: FileKind): SourceReader | undefined =>
    stands.kind === 'book' && stands.format === 'epub'
      ? (turning ?? sources.get('book'))
      : sources.get(stands.kind)

  /**
   * A file just made here, put in front of the person as what it was made as.
   * The vault is not asked what stands there.
   */
  const made = (
    path: string,
    title: string,
    type: EditorKind,
    showing: PlexShowing = 'here',
    line?: number,
  ): void => {
    editors.get(type)?.(path, title, showing, line)
  }

  /**
   * A file put in front of the person in the editor made for what it is, in the
   * reader where a document stands there, or in the player where a recording
   * does. A path holding no source at all opens nothing.
   */
  const opens = async (
    path: string,
    title = '',
    showing: PlexShowing = 'here',
    line?: number,
  ): Promise<void> => {
    const stands = await fileKindAt(path)
    if (!stands) return
    if (stands.kind === 'note') return void made(path, title, stands.type, showing, line)
    readerOf(stands)?.(path, [])
  }

  /**
   * A source put in front of the person at spans of its own text: a book in
   * the reader and a recording in the player, at the first of them. A stretch
   * of a note's bytes names no line for the keyboard to stand on, so a note
   * opens whole.
   */
  const opensAt = async (path: string, spans: readonly Span[]): Promise<void> => {
    const stands = await fileKindAt(path)
    if (!stands) return
    if (stands.kind === 'note') return void made(path, '', stands.type)
    readerOf(stands)?.(path, spans)
  }

  return { holds, reads, turns, hears, points, opens, opensAt, made }
}

/** What the window puts files in front of the person with. */
export type FileOpeners = ReturnType<typeof fileOpeners>

/** Which of the four a file is made as. */
export type MakeKind = 'deck' | 'stencil' | 'preset' | 'url'

/** What the window asks the vault to make from nothing. */
export interface VaultMaker {
  /** A deck of no cards, filed in that folder under a name made from the title. */
  makeDeck(title: string, folder: string): Promise<MakeResult>
  /**
   * A stencil declaring those fields and showing no face, the same way. The
   * first field names the cards it cuts.
   */
  makeStencil(title: string, folder: string, fields: readonly string[]): Promise<MakeResult>
  /** A preset naming none of its settings, the same way. */
  makesPreset(title: string, folder: string): Promise<MakeResult>
  /**
   * A note pointing at an address, named by the address. What is at it is
   * fetched afterwards, and says what the note is called from then on.
   */
  makesURL(address: string, folder: string): Promise<MakeResult>
}

/** Everything making one of the four says in the window's voice. */
export interface MakeWords {
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<RefusalReason, string>
  /** What the one field a stencil is made carrying is called. */
  readonly field: string
}

/**
 * A deck, a stencil or a preset made in a folder under the name it is given.
 * The vault names the file after it and answers where it stands. A vault that
 * answers nothing at all is said here, because the roads that ask for one carry
 * no word of their own.
 */
export function fileMakers(vault: VaultMaker, puts: FileOpeners, words: MakeWords, said: MessageWriter) {
  const makes = async (what: MakeKind, folder: string, name: string): Promise<string> => {
    try {
      const answer =
        what === 'deck'
          ? await vault.makeDeck(name, folder)
          : what === 'stencil'
            ? await vault.makeStencil(name, folder, [words.field])
            : what === 'url'
              ? await vault.makesURL(name, folder)
              : await vault.makesPreset(name, folder)
      if (answer.refusal) {
        said(words.refused[answer.refusal], 'refusal')
        return ''
      }
      return answer.path
    } catch (error) {
      said(troubleWords(error), 'refusal')
      return ''
    }
  }

  /** The same, put in front of the person in a tab of its own. */
  const opens = async (what: MakeKind, folder: string, name: string): Promise<string> => {
    const path = await makes(what, folder, name)
    if (!path) return ''
    puts.made(path, '', what)
    return path
  }

  return {
    makes,
    decks: (folder: string, name: string) => opens('deck', folder, name),
    stencils: (folder: string, name: string) => opens('stencil', folder, name),
    presets: (folder: string, name: string) => opens('preset', folder, name),
    imports: (folder: string, address: string) => opens('url', folder, address),
  }
}
