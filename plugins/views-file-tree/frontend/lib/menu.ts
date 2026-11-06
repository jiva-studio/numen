/**
 * The menu on a row, drawn from the entries of the slot `fileTree.context`: a row per command,
 * view and submenu, and a new group after each separator.
 */
import type { MenuItem } from '@numen/ui'
import type { CommandChoice, SlotCommand, SlotItem, SlotSubmenu, SlotView } from '@numen/plugin-sdk'

/** The command of the window that moves files to the trash, run by the Delete key of the tree. */
export const TRASH_NOTE = 'app:note.trash'

/** An entry the menu draws as a row. */
export type MenuRowEntry = SlotCommand | SlotView | SlotSubmenu

/** The choices read for the commands of the open menu, by the key of their entry. */
export interface MenuChoices {
  /** The choices of a command; null while they are read. */
  readonly byKey: ReadonlyMap<string, readonly CommandChoice[] | null>
  /** The row of a submenu whose choices are being read. */
  readonly loadingText: string
  /** The row of a submenu of a command that offers no choice. */
  readonly noneText: string
}

const CHOICE_MARK = '#choice:'

/** The id of the row of a choice of the command of an entry. */
const getChoiceRowId = (key: string, choice: string): string => `${key}${CHOICE_MARK}${choice}`

/** The key of the entry and the id of the choice a row of a choice stands for; null for any other row. */
export const parseChoiceRowId = (id: string): { key: string; choice: string } | null => {
  const markIndex = id.indexOf(CHOICE_MARK)
  return markIndex < 0
    ? null
    : { key: id.slice(0, markIndex), choice: id.slice(markIndex + CHOICE_MARK.length) }
}

/** The rows of the submenu of a command that offers choices. */
const createChoiceItems = (entry: SlotCommand, choices: MenuChoices | undefined): MenuItem[] => {
  const read = choices?.byKey.get(entry.key)
  if (!read) {
    return [{ id: `${entry.key}#loading`, text: choices?.loadingText ?? '', isDisabled: true }]
  }
  if (read.length === 0) {
    return [{ id: `${entry.key}#none`, text: choices?.noneText ?? '', isDisabled: true }]
  }
  return read.map((choice) => ({ id: getChoiceRowId(entry.key, choice.id), text: choice.title }))
}

/** The commands of the entries that offer choices, inside submenus too. */
export const listChoiceCommands = (entries: readonly SlotItem[]): readonly SlotCommand[] =>
  entries.flatMap((entry) => {
    if (entry.kind === 'command') return entry.hasChoices ? [entry] : []
    return entry.kind === 'submenu' ? listChoiceCommands(entry.entries) : []
  })

/** The rows of the entries, in their order; the rows between two separators share a group. */
export const createMenuItems = (
  entries: readonly SlotItem[],
  choices?: MenuChoices,
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
      ...(entry.kind === 'submenu' ? { children: createMenuItems(entry.entries, choices) } : {}),
      ...(entry.kind === 'command' && entry.hasChoices
        ? { children: createChoiceItems(entry, choices) }
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
