/**
 * What a file of the vault is put in front of the person with.
 *
 * One window decides this once, and every road to a file comes through here: a
 * road holds a path and no choice. What the vault holds at the path decides
 * between the editors and the reader, and the vault is asked what that is; a
 * kind of tab hands over the way it opens a file and keeps none of its own.
 */
import type { PlexShowing } from '@numen/ui'
import type { NoteType, Run, Standing } from './core'

/**
 * What a file the window opens is opened as: which of three a note is, or the
 * preset a fourth kind of note holds.
 */
export type Opened = NoteType | 'preset'

/**
 * A file put in front of the person in one editor. A line is somewhere inside
 * the file, and what standing there comes to is the editor's own.
 */
export type Opens = (
  path: string,
  title: string,
  showing: PlexShowing,
  line?: number,
) => void

/**
 * A source put in front of the person in the reader. The stretches are of the
 * source's own text, and the person is taken to the first of them.
 */
export type Reads = (path: string, runs: readonly Run[]) => void

/** What the window asks the vault about the file it is opening. */
export interface Asking {
  standing(paths: readonly string[]): Promise<ReadonlyMap<string, Standing>>
}

/** What a file the vault could not be asked about at all is opened as. */
const ORDINARY: Standing = { kind: 'note', type: 'note' }

export function putting(vault: Asking) {
  /** The editor each kind of note opens in, as its kind handed it over. */
  const editors = new Map<Opened, Opens>()
  /** The reader a source that is not a note opens in, handed over the same way. */
  let reader: Reads | null = null

  /** A kind of tab hands over the way it puts a file in front of the person. */
  const holds = (type: Opened, opens: Opens) => {
    editors.set(type, opens)
  }

  /** The kind of tab that reads documents hands its own over. */
  const reads = (opens: Reads) => {
    reader = opens
  }

  /**
   * What stands at a path. Nothing stands where the vault answers nothing; a
   * vault that cannot be asked at all leaves it the ordinary note it reads as.
   */
  const standingAt = async (path: string): Promise<Standing | null> => {
    try {
      return (await vault.standing([path])).get(path) ?? null
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
   * A file put in front of the person in the editor made for what it is, or in
   * the reader where a document stands there. A path holding no source at all
   * opens nothing.
   */
  const opens = async (
    path: string,
    title = '',
    showing: PlexShowing = 'here',
    line?: number,
  ): Promise<void> => {
    const stands = await standingAt(path)
    if (!stands) return
    if (stands.kind === 'book') return void reader?.(path, [])
    if (stands.kind === 'note') made(path, title, stands.type, showing, line)
  }

  /**
   * A source put in front of the person at stretches of its own text: a book in
   * the reader, at the first of them. A stretch of a note's bytes names no line
   * for the keyboard to stand on, so a note opens whole.
   */
  const opensAt = async (path: string, runs: readonly Run[]): Promise<void> => {
    const stands = await standingAt(path)
    if (!stands) return
    if (stands.kind === 'book') return void reader?.(path, runs)
    if (stands.kind === 'note') made(path, '', stands.type)
  }

  return { holds, reads, opens, opensAt, made }
}

/** What the window puts files in front of the person with. */
export type Putting = ReturnType<typeof putting>
