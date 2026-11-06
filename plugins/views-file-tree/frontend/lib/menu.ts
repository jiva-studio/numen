/**
 * The menu on a row, drawn from the entries of the slot `fileTree.context`: a row per command,
 * view and submenu, and a new group after each separator.
 */
import type { MenuItem } from '@numen/ui'
import type { CommandItem, SlotCommand, SlotItem, SlotSubmenu, SlotView } from '@numen/plugin-sdk'

/** The command of the window that moves files to the trash, run by the Delete key of the tree. */
export const TRASH_NOTE = 'app:note.trash'

/** An entry the menu draws as a row. */
export type MenuRowEntry = SlotCommand | SlotView | SlotSubmenu

/** The items read for the commands of the open menu, by the key of their entry. */
export interface MenuItemLists {
  /** The items of a command; null while they are read. */
  readonly byKey: ReadonlyMap<string, readonly CommandItem[] | null>
  /** The row of a submenu whose items are being read. */
  readonly loadingText: string
  /** The row of a submenu of a command that lists no item. */
  readonly noneText: string
}

const ITEM_MARK = '#item:'

/** The id of the row of an item of the command of an entry. */
const getItemRowId = (key: string, item: string): string => `${key}${ITEM_MARK}${item}`

/** The key of the entry and the id of the item a row of an item stands for; null for any other row. */
export const parseItemRowId = (id: string): { key: string; item: string } | null => {
  const markIndex = id.indexOf(ITEM_MARK)
  return markIndex < 0
    ? null
    : { key: id.slice(0, markIndex), item: id.slice(markIndex + ITEM_MARK.length) }
}

/** The rows of the submenu of a command that offers items. */
const createItemRows = (entry: SlotCommand, lists: MenuItemLists | undefined): MenuItem[] => {
  const read = lists?.byKey.get(entry.key)
  if (!read) {
    return [{ id: `${entry.key}#loading`, text: lists?.loadingText ?? '', isDisabled: true }]
  }
  if (read.length === 0) {
    return [{ id: `${entry.key}#none`, text: lists?.noneText ?? '', isDisabled: true }]
  }
  return read.map((item) => ({ id: getItemRowId(entry.key, item.id), text: item.title }))
}

/** The commands of the entries that offer items, inside submenus too. */
export const listItemCommands = (entries: readonly SlotItem[]): readonly SlotCommand[] =>
  entries.flatMap((entry) => {
    if (entry.kind === 'command') return entry.hasItems ? [entry] : []
    return entry.kind === 'submenu' ? listItemCommands(entry.entries) : []
  })

/** The rows of the entries, in their order; the rows between two separators share a group. */
export const createMenuItems = (
  entries: readonly SlotItem[],
  lists?: MenuItemLists,
): readonly MenuItem[] => {
  let group = 0
  const items: MenuItem[] = []
  for (const entry of entries) {
    if (entry.kind === 'separator') {
      group += 1
      continue
    }
    const groupName = String(group)
    if (entry.kind === 'view') {
      const component = entry.component as NonNullable<MenuItem['component']>
      items.push({ id: entry.key, text: entry.view, group: groupName, component })
      continue
    }
    items.push({
      id: entry.key,
      text: entry.title,
      group: groupName,
      ...(entry.kind === 'submenu' ? { children: createMenuItems(entry.entries, lists) } : {}),
      ...(entry.kind === 'command' && entry.hasItems
        ? { children: createItemRows(entry, lists) }
        : {}),
    })
  }
  return items
}

/** The entry drawn as the row `id`, inside a submenu too; undefined for an id no row carries. */
export const findMenuEntry = (
  entries: readonly SlotItem[],
  id: string,
): MenuRowEntry | undefined => {
  for (const entry of entries) {
    if (entry.kind === 'separator') continue
    if (entry.key === id) return entry
    if (entry.kind === 'submenu') {
      const found = findMenuEntry(entry.entries, id)
      if (found) return found
    }
  }
  return undefined
}
