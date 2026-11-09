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

/**
 * The entries of a slot in drawing order: those its layout lists, in the order of their first place
 * in the list, then those placed there that no slot lists, by group, order and plugin id.
 */
export function sortPlaced(
  placed: readonly Placed[],
  getLayoutIndex: (entry: DeclaredSlotEntry) => number | undefined,
): Placed[] {
  const inLayout: { readonly one: Placed; readonly index: number }[] = []
  const outsideLayout: Placed[] = []
  for (const one of keepFirstOfEachCommand(placed)) {
    const index = getLayoutIndex(one.entry)
    if (index === undefined) outsideLayout.push(one)
    else inLayout.push({ one, index })
  }
  return [
    ...inLayout
      .sort((a, b) => a.index - b.index || compareInSlot(a.one, b.one))
      .map(({ one }) => one),
    ...outsideLayout.sort(compareInSlot),
  ]
}

/** The items of a slot: its entries in drawing order, with a separator between two neighbours of different groups. */
export function layOut(slot: string, ordered: readonly Placed[]): SlotItem[] {
  const items: SlotItem[] = []
  let separators = 0
  ordered.forEach((one, index) => {
    const previous = ordered[index - 1]
    if (previous && (previous.entry.group ?? '') !== (one.entry.group ?? '')) {
      items.push({ kind: 'separator', key: `${slot}/separator/${separators}` })
      separators += 1
    }
    items.push(one.item)
  })
  return items
}
