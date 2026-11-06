/**
 * The commands of the window: which ones it runs now, the choices a command offers, and the one way
 * a plugin runs a command, its own or another's, from a button, a menu row or its code.
 */
import type { Disposable } from './types'

// CommandTarget is what a command runs over: the file or folder a row of the file tree was opened
// on, the file of a tab, the note of a search result or the note a plugin names. A bar has none.
export interface CommandTarget {
  // The path of the file or folder; empty where there is none.
  readonly path: string
  // What the target is called, such as the title of a note; empty where there is none.
  readonly title: string
  readonly isFolder: boolean
  // The other paths selected together with the target, as in the file tree.
  readonly others: readonly string[]
  // The id of the tab the command runs over, as in a tab menu; empty where there is none.
  readonly tab: string
}

// CommandChoice is one choice a command offers over a target, such as one open window for "Move to
// window". A place draws the command as a submenu of its choices, the palette as a pick list.
export interface CommandChoice {
  readonly id: string
  readonly title: string
}

// CommandsHost is the commands of the window.
export interface CommandsHost {
  // The full ids of the commands the window runs now (`<plugin id>:<id>`, `app:<id>`): its own and
  // those of the enabled plugins it has started.
  getIds(): readonly string[]
  // Calls the handler with the ids at once and again whenever they change, as plugins start and stop.
  watch(handler: (ids: readonly string[]) => void): Disposable
  // The choices the command offers over the target; empty for a command without choices. A place
  // shows a loading row while the list is pending.
  getChoices(command: string, target?: CommandTarget): Promise<readonly CommandChoice[]>
  // Runs the command of the full id over the target; with `choice`, the id of the choice picked.
  run(command: string, target?: CommandTarget, choice?: string): Promise<void>
}
