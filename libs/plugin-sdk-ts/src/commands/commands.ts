/**
 * The commands of the window: which ones it runs now, the items a command offers, and the one way
 * a plugin runs a command, its own or another's, from a button, a menu row or its code.
 */
import type { Disposable } from '../types'

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
  // The type the vault files the note of the target as, which a place reads before it asks for the
  // entries of its slot; empty or absent where the place has not read it or the target is no note.
  readonly noteType?: string
  // The frontmatter of the note of the target by its keys, read with its note type, which conditions
  // read as target.properties; absent where the place has not read it or the target is no note.
  readonly properties?: Readonly<Record<string, unknown>>
}

// CommandItem is one item a command offers over a target, such as one open window for "Move to
// window". A place draws the command as a submenu of its items, the palette as a pick list.
export interface CommandItem {
  readonly id: string
  readonly title: string
  // The plugin whose handler of a shared command offers the item, where the handlers offer items of
  // their own. A place draws the items of one group together and a separator between two groups.
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
  // Runs the command of the full id over the target; with `argument`, the id of the item picked. A
  // shared command without an argument runs its one handler whose condition holds over the target,
  // and runs nothing where several or none do.
  run(command: string, target?: CommandTarget, argument?: string): Promise<void>
}
