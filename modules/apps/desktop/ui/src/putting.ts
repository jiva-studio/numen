/**
 * What a file of the vault is put in front of the person with.
 *
 * One window decides this once, and every road to a file comes through here: a
 * road holds a path and no choice. Which of three the note is decides the
 * editor, and the vault is asked which; a kind of tab hands over the way it
 * opens a file and keeps none of its own.
 */
import type { PlexShowing } from '@numen/ui'
import type { NoteType } from './core'

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

/** What the window asks the vault about the file it is opening. */
export interface Asking {
  types(paths: readonly string[]): Promise<ReadonlyMap<string, NoteType>>
}

/** How a file the vault says nothing about is opened. */
const ORDINARY: NoteType = 'note'

export function putting(vault: Asking) {
  /** The editor each of three opens in, as its kind handed it over. */
  const editors = new Map<NoteType, Opens>()

  /** A kind of tab hands over the way it puts a file in front of the person. */
  const holds = (type: NoteType, opens: Opens) => {
    editors.set(type, opens)
  }

  /**
   * Which of three the note at a path is. A vault that cannot say leaves it the
   * ordinary note it reads as.
   */
  const typeOf = async (path: string): Promise<NoteType> => {
    try {
      return (await vault.types([path])).get(path) ?? ORDINARY
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
    type: NoteType,
    showing: PlexShowing = 'here',
    line?: number,
  ): void => {
    editors.get(type)?.(path, title, showing, line)
  }

  /** A file put in front of the person in the editor made for what it is. */
  const opens = async (
    path: string,
    title = '',
    showing: PlexShowing = 'here',
    line?: number,
  ): Promise<void> => {
    made(path, title, await typeOf(path), showing, line)
  }

  return { holds, opens, made }
}

/** What the window puts files in front of the person with. */
export type Putting = ReturnType<typeof putting>
