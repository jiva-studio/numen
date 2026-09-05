/**
 * What a file of the vault is put in front of the person with.
 *
 * Every road to a file comes through here, holding a path and no choice: what
 * the vault says it holds there decides between the editors, the reader and the
 * player. A kind of tab hands over the way it opens a file and keeps none.
 */
import type { PlexShowing } from '@numen/ui'
import type { FileKind, Made, NoteType, Refused, Stretch } from './core'
import type { Voice } from './telling'

/**
 * What a file the window opens is opened as: which of three a note is, or the
 * preset a fourth kind of note holds.
 */
export type Opened = NoteType | 'preset'

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
 * stretches are of the source's own text, and the person is taken to the first
 * of them.
 */
export type SourceReader = (path: string, stretches: readonly Stretch[]) => void

/** What the window asks the vault about the file it is opening. */
export interface PuttingDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
}

/** What a file the vault could not be asked about at all is opened as. */
const ORDINARY: FileKind = { kind: 'note', type: 'note' }

export function putting(vault: PuttingDeps) {
  /** The editor each kind of note opens in, as its kind handed it over. */
  const editors = new Map<Opened, FileOpener>()

  /**
   * The reader a document opens in and the player a recording is heard in, each
   * under the kind the vault answers a path with.
   */
  const sources = new Map<FileKind['kind'], SourceReader>()

  /** A kind of tab hands over the way it puts a file in front of the person. */
  const holds = (type: Opened, opens: FileOpener) => {
    editors.set(type, opens)
  }

  /** The kind of tab that reads documents hands its own over. */
  const reads = (opens: SourceReader) => {
    sources.set('book', opens)
  }

  /** The kind of tab that plays recordings hands its own over. */
  const hears = (opens: SourceReader) => {
    sources.set('recording', opens)
  }

  /**
   * What stands at a path. Nothing stands where the vault answers nothing; a
   * vault that cannot be asked at all leaves it the ordinary note it reads as.
   */
  const fileKindAt = async (path: string): Promise<FileKind | null> => {
    try {
      return (await vault.fileKinds([path])).get(path) ?? null
    } catch {
      return ORDINARY
    }
  }

  /**
   * A file just made here, put in front of the person as what it was made as.
   * The vault is not asked what stands there.
   */
  const made = (
    path: string,
    title: string,
    type: Opened,
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
    sources.get(stands.kind)?.(path, [])
  }

  /**
   * A source put in front of the person at stretches of its own text: a book in
   * the reader and a recording in the player, at the first of them. A stretch
   * of a note's bytes names no line for the keyboard to stand on, so a note
   * opens whole.
   */
  const opensAt = async (path: string, stretches: readonly Stretch[]): Promise<void> => {
    const stands = await fileKindAt(path)
    if (!stands) return
    if (stands.kind === 'note') return void made(path, '', stands.type)
    sources.get(stands.kind)?.(path, stretches)
  }

  return { holds, reads, hears, opens, opensAt, made }
}

/** What the window puts files in front of the person with. */
export type Putting = ReturnType<typeof putting>

/** Which of the three a file is made as. */
export type Cut = 'deck' | 'stencil' | 'preset'

/** What the window asks the vault to make from nothing. */
export interface CutWriter {
  /** A deck of no cards, filed in that folder under a name made from the title. */
  makeDeck(title: string, folder: string): Promise<Made>
  /**
   * A stencil declaring those fields and showing no face, the same way. The
   * first field names the cards it cuts.
   */
  makeStencil(title: string, folder: string, fields: readonly string[]): Promise<Made>
  /** A preset naming none of its settings, the same way. */
  makesPreset(title: string, folder: string): Promise<Made>
}

/** Everything making one of the three says in the window's voice. */
export interface CuttingWords {
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<Refused, string>
  /** What the one field a stencil is made carrying is called. */
  readonly field: string
}

/**
 * A deck, a stencil or a preset made in a folder under the name it is given.
 * The vault names the file after it and answers where it stands. A vault that
 * answers nothing at all is said here, because the roads that ask for one carry
 * no word of their own.
 */
export function cutting(vault: CutWriter, puts: Putting, words: CuttingWords, said: Voice) {
  const makes = async (what: Cut, folder: string, name: string): Promise<string> => {
    try {
      const answer =
        what === 'deck'
          ? await vault.makeDeck(name, folder)
          : what === 'stencil'
            ? await vault.makeStencil(name, folder, [words.field])
            : await vault.makesPreset(name, folder)
      if (answer.refusal) {
        said(words.refused[answer.refusal], 'refusal')
        return ''
      }
      return answer.path
    } catch (error) {
      said(String(error), 'refusal')
      return ''
    }
  }

  /** The same, put in front of the person in a tab of its own. */
  const opens = async (what: Cut, folder: string, name: string): Promise<string> => {
    const path = await makes(what, folder, name)
    if (!path) return ''
    puts.made(path, '', what)
    return path
  }

  return {
    makes,
    cuts: (folder: string, name: string) => opens('deck', folder, name),
    stencils: (folder: string, name: string) => opens('stencil', folder, name),
    presets: (folder: string, name: string) => opens('preset', folder, name),
  }
}
