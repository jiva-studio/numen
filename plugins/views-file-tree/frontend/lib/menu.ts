/**
 * The menu on a row, drawn from the entries of the slot `fileTree.context`: a row per command,
 * view and submenu, and a new group after each separator.
 */
import type { MenuItem } from '@numen/ui'
import type { SlotCommand, SlotItem, SlotSubmenu, SlotView } from '@numen/plugin-sdk'

/** The command of the window that moves files to the trash, run by the Delete key of the tree. */
export const TRASH_NOTE = 'app:note.trash'

/** An entry the menu draws as a row. */
export type MenuRowEntry = SlotCommand | SlotView | SlotSubmenu

/** The rows of the entries, in their order; the rows between two separators share a group. */
export const createMenuItems = (entries: readonly SlotItem[]): readonly MenuItem[] => {
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
      ...(entry.kind === 'submenu' ? { children: createMenuItems(entry.entries) } : {}),
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
