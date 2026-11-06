/**
 * The rows of the entries of the slot `tabStrip.context`: a row per command, view and submenu, a
 * submenu of items for a command that offers them, and a new group after each separator.
 */
import type { Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import type { CommandItem, SlotCommand, SlotItem } from '@numen/plugin-host'

/** The prefix of the group of the entries of the slot, kept apart from the groups of the rows. */
const SLOT_GROUP = 'slot:'

const ITEM_MARK = '#item:'

export type SlotRowEntry = Exclude<SlotItem, { kind: 'separator' }>

/** The items read for the commands of the open menu, by the key of their entry. */
export interface TabMenuItemLists {
  /** The items of a command; null while they are read. */
  readonly byKey: ReadonlyMap<string, readonly CommandItem[] | null>
  readonly loadingText: string
  readonly noneText: string
}

/** The key of the entry and the id of the item a row of an item stands for; null otherwise. */
export const parseItemRowId = (id: string): { key: string; item: string } | null => {
  const markIndex = id.indexOf(ITEM_MARK)
  return markIndex < 0
    ? null
    : { key: id.slice(0, markIndex), item: id.slice(markIndex + ITEM_MARK.length) }
}

/** The rows of the submenu of a command that offers items. */
const createItemRows = (entry: SlotCommand, lists: TabMenuItemLists | undefined): MenuItem[] => {
  const read = lists?.byKey.get(entry.key)
  if (!read) {
    return [{ id: `${entry.key}#loading`, text: lists?.loadingText ?? '', isDisabled: true }]
  }
  if (read.length === 0) {
    return [{ id: `${entry.key}#none`, text: lists?.noneText ?? '', isDisabled: true }]
  }
  return read.map((item) => ({
    id: `${entry.key}${ITEM_MARK}${item.id}`,
    text: item.title,
  }))
}

/** The commands of the entries that offer items, inside submenus too. */
export const listItemCommands = (entries: readonly SlotItem[]): readonly SlotCommand[] =>
  entries.flatMap((entry) => {
    if (entry.kind === 'command') return entry.hasItems ? [entry] : []
    return entry.kind === 'submenu' ? listItemCommands(entry.entries) : []
  })

/** The rows of the entries of the slot; the rows between two separators share a group. */
export const createSlotItems = (
  entries: readonly SlotItem[],
  lists?: TabMenuItemLists,
  group = 0,
): readonly MenuItem[] => {
  const items: MenuItem[] = []
  for (const entry of entries) {
    if (entry.kind === 'separator') {
      group += 1
      continue
    }
    const groupName = `${SLOT_GROUP}${group}`
    if (entry.kind === 'view') {
      const component = entry.component as Component
      items.push({ id: entry.key, text: entry.view, group: groupName, component })
      continue
    }
    items.push({
      id: entry.key,
      text: entry.title,
      group: groupName,
      ...(entry.kind === 'submenu'
        ? { children: createSlotItems(entry.entries, lists, group) }
        : {}),
      ...(entry.kind === 'command' && entry.hasItems
        ? { children: createItemRows(entry, lists) }
        : {}),
    })
  }
  return items
}

/** The entry drawn as the row `id`, inside a submenu too. */
export const findSlotEntry = (
  entries: readonly SlotItem[],
  id: string,
): SlotRowEntry | undefined => {
  for (const entry of entries) {
    if (entry.kind === 'separator') continue
    if (entry.key === id) return entry
    if (entry.kind === 'submenu') {
      const found = findSlotEntry(entry.entries, id)
      if (found) return found
    }
  }
  return undefined
}
