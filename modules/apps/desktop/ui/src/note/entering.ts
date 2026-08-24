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
   * until there is an editor to hand it to. Each is under the identity its tab
   * opened with, which it keeps wherever its file goes.
   */
  const owed = new Map<string, number>()

  /** The editor of each open note, for as long as its tab is drawn. */
  const editors = new Map<string, Drawn>()

  const enters = (id: string) => {
    const line = owed.get(id)
    const editor = editors.get(id)
    if (line === undefined || !editor) return
    // An editor is registered as it is drawn, a moment before it exists to take
    // anything. The note is owed its keyboard until one has.
    if (line >= 0 ? editor.reveal(line) : editor.focus()) owed.delete(id)
  }

  /** A note owed the keyboard, on the line it is to stand on. */
  const owes = (id: string, line = ITSELF) => {
    owed.set(id, line)
    void nextTick(() => enters(id))
  }

  /** The editor of one note, as it is drawn and as it goes. */
  const drew = (id: string, editor: unknown) => {
    if (!editor) {
      editors.delete(id)
      return
    }
    editors.set(id, editor as Drawn)
    void nextTick(() => enters(id))
  }

  /** The note is on screen: its editor measures, and takes what it is owed. */
  const measure = (id: string) => {
    editors.get(id)?.measure()
    enters(id)
  }

  /** The note is closing, and is owed nothing more. */
  const drops = (id: string) => owed.delete(id)

  return { owes, drew, measure, drops }
}
