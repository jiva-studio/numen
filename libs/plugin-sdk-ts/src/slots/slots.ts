/**
 * The slots of a window as a plugin reads and fills them. The manifest places entries with
 * `contributes.slots`; the owner of a slot reads them here, draws them, and runs their commands over
 * its target.
 */
import type { Disposable } from '../types'
import type { SlotName } from './slotNames'

export { SLOT, WINDOW_SLOTS, type SlotName } from './slotNames'
export { WHEN_KEYS, type WhenKey } from './whenKeys'

// SlotPlace is a slot of the window or a submenu a plugin declares; a submenu is named by its id.
export type SlotPlace = SlotName | (string & {})

// CommandTarget is what a command runs over: the file or folder a row of the file tree was opened
// on, the file of a tab, the note of a search result. A bar has none.
export interface CommandTarget {
  // The path of the file or folder; empty where there is none.
  readonly path: string
  // What the target is called, such as the title of a note; empty where there is none.
  readonly title: string
  readonly isFolder: boolean
  // The other paths selected together with the target, as in the file tree.
  readonly others: readonly string[]
}

// SlotEntryKey names one entry for as long as its plugin declares it: the plugin and its command,
// view or submenu, as `<plugin id>/<kind>/<id>`.
export type SlotEntryKey = string

// SlotCommand is a command the owner of the slot draws: a button in a bar, a row in a menu.
export interface SlotCommand {
  readonly kind: 'command'
  readonly key: SlotEntryKey
  // The plugin that placed the entry.
  readonly pluginId: string
  // The full id of the command, which `SlotHost.runCommand` takes.
  readonly command: string
  readonly title: string
  // The icon component of the command; undefined for a command without one.
  readonly icon?: unknown
  // The suggested shortcut of the command, such as `mod+shift+e`; empty for none.
  readonly shortcut: string
}

// SlotView is a component a plugin registered with `registerSlotView` and draws itself.
export interface SlotView {
  readonly kind: 'view'
  readonly key: SlotEntryKey
  readonly pluginId: string
  // The id the manifest places the view under.
  readonly view: string
  readonly component: unknown
}

// SlotSubmenu is a submenu a plugin declares, holding the entries that name it as their slot.
export interface SlotSubmenu {
  readonly kind: 'submenu'
  readonly key: SlotEntryKey
  readonly pluginId: string
  readonly submenu: string
  readonly title: string
  // The icon component of the submenu; undefined for one without.
  readonly icon?: unknown
  // The entries of the submenu, resolved by the same rules as a slot; never empty.
  readonly entries: readonly SlotItem[]
}

// SlotSeparator divides two groups of a slot: a line in a menu, a gap in a bar. Never first,
// last, or next to another.
export interface SlotSeparator {
  readonly kind: 'separator'
  readonly key: SlotEntryKey
}

// SlotItem is one thing the owner of a slot draws, in order.
export type SlotItem = SlotCommand | SlotView | SlotSubmenu | SlotSeparator

// SlotHost is the slots of the window as their owners read them.
export interface SlotHost {
  // The entries of the slot over the target, in the order they are drawn, groups resolved into
  // separators. An entry whose command is not available over the target or in this window, or whose
  // plugin is suspended, is left out; so is a view that has not been registered yet.
  getEntries(slot: SlotPlace, target?: CommandTarget): readonly SlotItem[]
  // Calls the handler with the entries of the slot at once and again on every change.
  watchEntries(
    slot: SlotPlace,
    handler: (entries: readonly SlotItem[]) => void,
    target?: CommandTarget,
  ): Disposable
}
