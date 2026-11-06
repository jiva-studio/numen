/**
 * The commands of the window: which ones it runs now, and the one way a plugin runs a command, its
 * own or another's, from a button, a menu row or its code.
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
}

// CommandsHost is the commands of the window.
export interface CommandsHost {
  // The full ids of the commands the window runs now (`<plugin id>:<id>`, `app:<id>`): its own and
  // those of the enabled plugins it has started.
  getIds(): readonly string[]
  // Calls the handler with the ids at once and again whenever they change, as plugins start and stop.
  watch(handler: (ids: readonly string[]) => void): Disposable
  // Runs the command of the full id over the target.
  run(command: string, target?: CommandTarget): Promise<void>
}
