/**
 * The slots of a window as a plugin reads and fills them. The manifest places entries with
 * `contributes.slots`; the owner of a slot reads them here, draws them, and runs their commands
 * over its target through `CommandsHost.run`.
 */
import type { Component } from 'vue'
import type { Disposable } from '../types'
import type { SlotName } from './slotNames'
import type { CommandTarget } from '../commands'

export { SLOT, type SlotName } from './slotNames'
export { WHEN_VARIABLES } from './whenVariables'

// SlotPlace is a slot of the window or a submenu a plugin declares; a submenu is named by its id.
export type SlotPlace = SlotName | (string & {})

// SlotEntryKey names one entry for as long as its plugin declares it: the plugin and its command,
// view or submenu, as `<plugin id>/<kind>/<id>`.
export type SlotEntryKey = string

// SlotCommand is a command the owner of the slot draws: a button in a bar, a row in a menu.
export interface SlotCommand {
  readonly kind: 'command'
  readonly key: SlotEntryKey
  readonly pluginId: string
  readonly command: string
  readonly title: string
  readonly icon?: Component
  readonly shortcut: string
  readonly hasItems: boolean
  readonly values?: Readonly<Record<string, string>>
}

// SlotView is a component a plugin registered with `registerSlotView` and draws itself.
export interface SlotView {
  readonly kind: 'view'
  readonly key: SlotEntryKey
  readonly pluginId: string
  readonly view: string
  readonly component: Component
}

// SlotSubmenu is a submenu a plugin declares, holding the entries that name it as their slot.
export interface SlotSubmenu {
  readonly kind: 'submenu'
  readonly key: SlotEntryKey
  readonly pluginId: string
  readonly submenu: string
  readonly title: string
  readonly icon?: Component
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
