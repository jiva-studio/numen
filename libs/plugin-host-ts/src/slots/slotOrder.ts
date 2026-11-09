/**
 * The order the entries of a slot are drawn in: groups sorted alphanumerically with the entries of no
 * group last, entries sorted by order and then plugin id, and one entry for each command.
 */
import type { SlotItem } from '@numen/plugin-sdk'
import type { DeclaredSlotEntry } from './slotRegistry'

export const compareText = (a: string, b: string): number => {
  if (a === b) return 0
  return a < b ? -1 : 1
}

/** An entry with the item it draws, in the order a slot sorts them. */
export interface Placed {
  readonly entry: DeclaredSlotEntry
  readonly item: SlotItem
}

const compareInGroup = (a: Placed, b: Placed): number => {
  if (a.entry.order !== b.entry.order) {
    if (a.entry.order === undefined) return 1
    if (b.entry.order === undefined) return -1
    return a.entry.order - b.entry.order
  }
  return compareText(a.entry.pluginId, b.entry.pluginId) || compareText(a.item.key, b.item.key)
}

const compareGroups = (a: string, b: string): number => {
  if (a === b) return 0
  if (a === '') return 1
  if (b === '') return -1
  return compareText(a, b)
}

export const compareInSlot = (a: Placed, b: Placed): number =>
  compareGroups(a.entry.group ?? '', b.entry.group ?? '') || compareInGroup(a, b)

export function keepFirstOfEachCommand(placed: readonly Placed[]): Placed[] {
  const first = new Map<string, Placed>()
  for (const one of placed) {
    if (one.item.kind !== 'command') continue
    const current = first.get(one.item.command)
    if (current === undefined || compareInSlot(one, current) < 0) first.set(one.item.command, one)
  }
  return placed.filter((one) => one.item.kind !== 'command' || first.get(one.item.command) === one)
}

/** The items of a slot that is not a bar: its groups in order, a separator between two of them. */
export function layOut(slot: string, placed: readonly Placed[]): SlotItem[] {
  const groups = new Map<string, Placed[]>()
  for (const one of keepFirstOfEachCommand(placed)) {
    const name = one.entry.group ?? ''
    groups.set(name, [...(groups.get(name) ?? []), one])
  }
  const items: SlotItem[] = []
  for (const name of [...groups.keys()].sort(compareGroups)) {
    if (items.length > 0) items.push({ kind: 'separator', key: `${slot}/separator/${name}` })
    items.push(...(groups.get(name) ?? []).sort(compareInGroup).map((one) => one.item))
  }
  return items
}
