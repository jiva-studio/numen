/**
 * The menu tree of the entries of a slot, as every owner of a slot draws it in a menu: a row per
 * command, view and submenu, a group between two separators, and a command with items as a submenu
 * of the items read for it.
 */
import type { Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import { isSharedCommand, type CommandItem } from '../commands'
import type { SlotCommand, SlotEntryKey, SlotItem } from './slots'

// CommandItemLists is the items read for the commands of an open menu, by the key of their entry;
// null while a list is being read.
export type CommandItemLists = ReadonlyMap<SlotEntryKey, readonly CommandItem[] | null>

// SlotMenuWords is the text of the rows a place writes itself.
export interface SlotMenuWords {
  readonly loadingText: string
}

// SlotMenuPick is what a row runs: a command, with the id of the item picked and the values its entry holds.
export interface SlotMenuPick {
  readonly command: string
  readonly itemId?: string
  readonly values?: Readonly<Record<string, string>>
}

// SlotMenu is the menu tree of the entries of a slot.
export interface SlotMenu {
  readonly items: readonly MenuItem[]
  // What the row of the id runs; undefined for a row that runs nothing.
  getPick(id: string): SlotMenuPick | undefined
  // The icon component of the entry the row of the id is drawn for; undefined for none.
  getIcon(id: string): Component | undefined
}

const ITEM_MARK = '#item:'

interface MenuParts {
  readonly lists: CommandItemLists
  readonly words: SlotMenuWords
  readonly picks: Map<string, SlotMenuPick>
  readonly icons: Map<string, Component>
}

const getValues = (entry: SlotCommand): Pick<SlotMenuPick, 'values'> =>
  entry.values === undefined ? {} : { values: entry.values }

function createItemsRow(entry: SlotCommand, group: string, parts: MenuParts): MenuItem | null {
  const list = parts.lists.get(entry.key)
  const row = { id: entry.key, text: entry.title, group }
  if (!list) {
    return {
      ...row,
      children: [{ id: `${entry.key}#loading`, text: parts.words.loadingText, isDisabled: true }],
    }
  }
  if (list.length === 0) return null
  const isShared = isSharedCommand(entry.command)
  const children = list.map((item): MenuItem => {
    const id = `${entry.key}${ITEM_MARK}${item.id}`
    parts.picks.set(id, { command: entry.command, itemId: item.id, ...getValues(entry) })
    return {
      id,
      text: item.title,
      ...(item.detail === undefined ? {} : { detail: item.detail }),
      ...(item.group === undefined ? {} : { group: item.group }),
    }
  })
  const [only] = children
  // A shared command with one handler is drawn as the row of that handler.
  if (isShared && only && list.length === 1 && list[0]?.group === undefined) {
    if (entry.icon !== undefined) parts.icons.set(only.id, entry.icon)
    return { ...only, group }
  }
  return { ...row, children }
}

function createRows(entries: readonly SlotItem[], parts: MenuParts): MenuItem[] {
  let group = 0
  const rows: MenuItem[] = []
  for (const entry of entries) {
    if (entry.kind === 'separator') {
      group += 1
      continue
    }
    const groupName = String(group)
    if (entry.kind === 'view') {
      rows.push({ id: entry.key, text: entry.view, group: groupName, component: entry.component })
      continue
    }
    if (entry.icon !== undefined) parts.icons.set(entry.key, entry.icon)
    if (entry.kind === 'submenu') {
      const children = createRows(entry.entries, parts)
      if (children.length > 0)
        rows.push({ id: entry.key, text: entry.title, group: groupName, children })
      continue
    }
    if (entry.hasItems) {
      const row = createItemsRow(entry, groupName, parts)
      if (row) rows.push(row)
      continue
    }
    parts.picks.set(entry.key, { command: entry.command, ...getValues(entry) })
    rows.push({ id: entry.key, text: entry.title, group: groupName })
  }
  return rows
}

/**
 * Builds the menu tree of the entries with the item lists read for their commands. A command with
 * items is a submenu of them, with a loading row while its list is read, and is hidden where it lists
 * none; a separator is drawn where the group of two items differs. A shared command with one handler
 * is the row of that handler. A submenu with no rows is hidden.
 */
export function createSlotMenu(
  entries: readonly SlotItem[],
  lists: CommandItemLists,
  words: SlotMenuWords,
): SlotMenu {
  const parts: MenuParts = { lists, words, picks: new Map(), icons: new Map() }
  const items = createRows(entries, parts)
  return {
    items,
    getPick: (id) => parts.picks.get(id),
    getIcon: (id) => parts.icons.get(id),
  }
}
