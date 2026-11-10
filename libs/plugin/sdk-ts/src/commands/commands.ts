/**
 * The commands of the window: which ones it runs now, the items a command offers, and the one way
 * a plugin runs a command, its own or another's, from a button, a menu row or its code.
 */
import type { JsonValue, Disposable } from '../types'
import { SHARED_COMMANDS } from './sharedCommands'

// isSharedCommand reports whether the full id is a shared command of the application.
export function isSharedCommand(id: string): boolean {
  return SHARED_COMMANDS.some((one) => one.id === id)
}

// CommandTarget is what a command runs over: the file or folder a row of the file tree was opened
// on, the file of a tab, the note of a search result or the note a plugin names. A run without
// one, as from a bar, is over the file of the tab in front.
export interface CommandTarget {
  readonly path: string
  readonly title: string
  readonly isFolder: boolean
  readonly others: readonly string[]
  readonly tab: string
  readonly noteType?: string
  readonly properties?: Readonly<Record<string, JsonValue>>
}

// CommandItem is one item a command offers over a target, such as one open window for "Move to
// window". A place draws the command as a submenu of its items, the palette as a pick list.
export interface CommandItem {
  readonly id: string
  readonly title: string
  readonly detail?: string
  readonly group?: string
}

// CommandsHost is the commands of the window.
export interface CommandsHost {
  // The full ids of the commands the window runs now (`<plugin id>:<id>`, `app:<id>`): its own and
  // those of the enabled plugins it has started.
  getIds(): readonly string[]
  // Calls the handler with the ids at once and again whenever they change, as plugins start and stop.
  watch(handler: (ids: readonly string[]) => void): Disposable
  // The items the command offers over the target; empty for a command without items. A place
  // shows a loading row while the list is pending. A shared command (SHARED_COMMANDS) offers its
  // handlers whose condition holds over the target, one item each with the plugin id as its id and
  // the handler's title; a place draws a single one as a row of that title. Where those handlers
  // offer items of their own, it offers theirs instead, in one list, each in the group of its plugin.
  listItems(command: string, target?: CommandTarget): Promise<readonly CommandItem[]>
  // Runs the command of the full id over the target; with `itemId`, the id of the item picked. A
  // shared command without an item id runs its one handler whose condition holds over the target,
  // and runs nothing where several or none do. Resolves with the path of the note or folder the run
  // made, empty where it made none. `values` are the values of the arguments of the command by name, as
  // text; a command without them runs as before.
  run(
    command: string,
    target?: CommandTarget,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): Promise<string>
}
