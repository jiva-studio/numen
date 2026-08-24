/**
 * The keyboard an open note is owed, and the editor that takes it.
 *
 * A note opened takes the keyboard once it is on screen, on the line it was
 * asked for when it was asked for one. A tab already showing has an editor
 * now; a tab that has to be drawn first says so when it appears, and an editor
 * says so when it is built.
 */
import { nextTick } from 'vue'

/** What the editor of a note answers once it is drawn. */
export interface Drawn {
  focus(): boolean
  measure(): void
  reveal(line: number): boolean
}

/**
 * The line a note is opened on where none was asked for: the note itself, and
 * no line in particular.
 */
export const ITSELF = -1

export function entering() {
  /**
   * The notes owed their keyboard, and the line each was asked to open on,
   * until there is an editor to hand it to.
   */
  const owed = new Map<string, number>()

  /** The editor of each open note, for as long as its tab is drawn. */
  const editors = new Map<string, Drawn>()

  const enters = (path: string) => {
    const line = owed.get(path)
    const editor = editors.get(path)
    if (line === undefined || !editor) return
    // An editor is registered as it is drawn, a moment before it exists to take
    // anything. The note is owed its keyboard until one has.
    if (line >= 0 ? editor.reveal(line) : editor.focus()) owed.delete(path)
  }

  /** A note owed the keyboard, on the line it is to stand on. */
  const owes = (path: string, line = ITSELF) => {
    owed.set(path, line)
    void nextTick(() => enters(path))
  }

  /** The editor of one note, as it is drawn and as it goes. */
  const drew = (path: string, editor: unknown) => {
    if (!editor) {
      editors.delete(path)
      return
    }
    editors.set(path, editor as Drawn)
    void nextTick(() => enters(path))
  }

  /** The note is on screen: its editor measures, and takes what it is owed. */
  const measure = (path: string) => {
    editors.get(path)?.measure()
    enters(path)
  }

  /** The note is closing, and is owed nothing more. */
  const drops = (path: string) => owed.delete(path)

  return { owes, drew, measure, drops }
}
