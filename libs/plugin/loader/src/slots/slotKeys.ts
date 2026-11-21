/** The keys a slot draws its entries under, and the order it draws them in when nothing is stored. */
import { formatCommandId } from '../commandId'
import type { DeclaredSlotEntry } from './slotRegistry'

export const compareText = (a: string, b: string): number => {
  if (a === b) return 0
  return a < b ? -1 : 1
}

export const compareOrder = (a: DeclaredSlotEntry, b: DeclaredSlotEntry): number => {
  if (a.order === b.order) return 0
  if (a.order === undefined) return 1
  if (b.order === undefined) return -1
  return a.order - b.order
}

export const compareGroups = (a: string, b: string): number => {
  if (a === b) return 0
  if (a === '') return 1
  if (b === '') return -1
  return compareText(a, b)
}

export const getFullCommand = (entry: DeclaredSlotEntry, command: string): string =>
  command.includes(':') ? command : formatCommandId(entry.pluginId, command)

export const getValuesKey = (values: DeclaredSlotEntry['values']): string => {
  const names = Object.keys(values ?? {}).sort(compareText)
  if (names.length === 0) return ''
  const pairs = names.map(
    (name) => `${encodeURIComponent(name)}=${encodeURIComponent(values?.[name] ?? '')}`,
  )
  return `?${pairs.join('&')}`
}

export const formatItemKey = (entry: DeclaredSlotEntry, kind: string, id: string): string =>
  `${entry.pluginId}/${kind}/${id}`

/**
 * The key a slot draws an entry under, which a layout names it by: `<plugin id>/view/<view>`,
 * `<plugin id>/command/<full command id>` followed by its values, or `<plugin id>/submenu/<id>`.
 */
export function getEntryKey(entry: DeclaredSlotEntry): string | undefined {
  if (entry.command) {
    const command = getFullCommand(entry, entry.command)
    return formatItemKey(entry, 'command', `${command}${getValuesKey(entry.values)}`)
  }
  if (entry.view) return formatItemKey(entry, 'view', entry.view)
  return entry.submenu ? formatItemKey(entry, 'submenu', entry.submenu) : undefined
}

/** The entries in the order a slot draws them when nothing is stored: by group, then order, then plugin id. */
export function sortSlotEntries(entries: readonly DeclaredSlotEntry[]): DeclaredSlotEntry[] {
  return [...entries].sort(
    (a, b) =>
      compareGroups(a.group ?? '', b.group ?? '') ||
      compareOrder(a, b) ||
      compareText(a.pluginId, b.pluginId) ||
      compareText(getEntryKey(a) ?? '', getEntryKey(b) ?? ''),
  )
}
