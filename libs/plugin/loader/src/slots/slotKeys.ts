/** The keys a slot draws its entries under, and the order it draws them in when nothing is stored. */
import { formatCommandId } from '../commandId'
import type { DeclaredSlotEntry } from './slotRegistry'

export const compareText = (a: string, b: string): number => {
  if (a === b) return 0
  return a < b ? -1 : 1
}

const compareOrder = (a: DeclaredSlotEntry, b: DeclaredSlotEntry): number => {
  if (a.order === b.order) return 0
  if (a.order === undefined) return 1
  if (b.order === undefined) return -1
  return a.order - b.order
}

const compareGroups = (a: string, b: string): number => {
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

const compareEntries = (a: DeclaredSlotEntry, b: DeclaredSlotEntry): number =>
  compareGroups(a.group ?? '', b.group ?? '') ||
  compareOrder(a, b) ||
  compareText(a.pluginId, b.pluginId) ||
  compareText(getEntryKey(a) ?? '', getEntryKey(b) ?? '')

const getRunKey = (entry: DeclaredSlotEntry): string =>
  entry.command ? `${getFullCommand(entry, entry.command)}${getValuesKey(entry.values)}` : ''

function keepFirstOfEachCommand<T>(
  items: readonly T[],
  getEntry: (one: T) => DeclaredSlotEntry,
): T[] {
  const first = new Map<string, T>()
  for (const one of items) {
    const entry = getEntry(one)
    if (!entry.command) continue
    const current = first.get(getRunKey(entry))
    if (current === undefined || compareEntries(entry, getEntry(current)) < 0)
      first.set(getRunKey(entry), one)
  }
  return items.filter(
    (one) => !getEntry(one).command || first.get(getRunKey(getEntry(one))) === one,
  )
}

/**
 * The entries of one slot as it draws them when nothing is stored: the first of each command, by group,
 * then order, then plugin id.
 */
export function sortSlotEntries<T>(
  items: readonly T[],
  getEntry: (one: T) => DeclaredSlotEntry,
): T[] {
  return keepFirstOfEachCommand(items, getEntry).sort((a, b) =>
    compareEntries(getEntry(a), getEntry(b)),
  )
}

/** What a slot in its default arrangement is made of: a row for each entry, a separator where the group changes. */
interface GroupedRows<T, R> {
  readonly group: (one: T) => string | undefined
  readonly buildRow: (one: T) => R
  readonly buildSeparator: (group: string) => R
}

/** The ordered entries as rows, with a separator before each group after the first, named by the group it opens. */
export function insertSeparators<T, R>(ordered: readonly T[], read: GroupedRows<T, R>): R[] {
  const rows: R[] = []
  let previous: string | undefined
  for (const one of ordered) {
    const group = read.group(one) ?? ''
    if (previous !== undefined && group !== previous) rows.push(read.buildSeparator(group))
    rows.push(read.buildRow(one))
    previous = group
  }
  return rows
}
