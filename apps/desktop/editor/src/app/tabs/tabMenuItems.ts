/**
 * The rows of the entries of the slot `tabStrip.context`: a row per command, view and submenu, a
 * submenu of choices for a command that offers them, and a new group after each separator.
 */
import type { Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import type { CommandChoice, SlotCommand, SlotItem } from '@numen/plugin-host'

/** The prefix of the group of the entries of the slot, kept apart from the groups of the rows. */
const SLOT_GROUP = 'slot:'

const CHOICE_MARK = '#choice:'

export type SlotRowEntry = Exclude<SlotItem, { kind: 'separator' }>

/** The choices read for the commands of the open menu, by the key of their entry. */
export interface TabMenuChoices {
  /** The choices of a command; null while they are read. */
  readonly byKey: ReadonlyMap<string, readonly CommandChoice[] | null>
  readonly loadingText: string
  readonly noneText: string
}

/** The key of the entry and the id of the choice a row of a choice stands for; null otherwise. */
export const parseChoiceRowId = (id: string): { key: string; choice: string } | null => {
  const markIndex = id.indexOf(CHOICE_MARK)
  return markIndex < 0
    ? null
    : { key: id.slice(0, markIndex), choice: id.slice(markIndex + CHOICE_MARK.length) }
}

/** The rows of the submenu of a command that offers choices. */
const createChoiceItems = (entry: SlotCommand, choices: TabMenuChoices | undefined): MenuItem[] => {
  const read = choices?.byKey.get(entry.key)
  if (!read) {
    return [{ id: `${entry.key}#loading`, text: choices?.loadingText ?? '', isDisabled: true }]
  }
  if (read.length === 0) {
    return [{ id: `${entry.key}#none`, text: choices?.noneText ?? '', isDisabled: true }]
  }
  return read.map((choice) => ({
    id: `${entry.key}${CHOICE_MARK}${choice.id}`,
    text: choice.title,
  }))
}

/** The commands of the entries that offer choices, inside submenus too. */
export const listChoiceCommands = (entries: readonly SlotItem[]): readonly SlotCommand[] =>
  entries.flatMap((entry) => {
    if (entry.kind === 'command') return entry.hasChoices ? [entry] : []
    return entry.kind === 'submenu' ? listChoiceCommands(entry.entries) : []
  })

/** The rows of the entries of the slot; the rows between two separators share a group. */
export const createSlotItems = (
  entries: readonly SlotItem[],
  choices?: TabMenuChoices,
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
        ? { children: createSlotItems(entry.entries, choices, group) }
        : {}),
      ...(entry.kind === 'command' && entry.hasChoices
        ? { children: createChoiceItems(entry, choices) }
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
