/**
 * The menu that moves an entry of a bar: `Move to`, a submenu for each slot the user arranges, and in it
 * one item for each entry drawn there, which puts the moved entry before it, and `At the end`.
 */
import type { MenuItem } from '@numen/ui'
import {
  moveSlotEntry,
  type LayoutEntry,
  type SlotLayout,
  type SlotLayoutChange,
  type SlotName,
} from '@numen/plugin-host'

/** Where a choice of the menu puts the moved entry: into the slot, before the key; at the end when there is none. */
export interface MoveTarget {
  readonly slot: SlotName
  readonly before?: string | undefined
}

export interface MoveMenuSource {
  readonly listSlots: () => readonly SlotName[]
  readonly listEntries: (slot: SlotName) => readonly LayoutEntry[]
  readonly getSlotTitle: (slot: SlotName) => string
}

export interface MoveMenuWords {
  readonly moveTo: string
  readonly atTheEnd: string
}

/** The items of the menu and, for each item that moves the entry, where it puts it. */
export interface MoveMenu {
  readonly items: readonly MenuItem[]
  readonly targets: ReadonlyMap<string, MoveTarget>
}

/** The menu of the entry `moved`; the entry is not listed among the entries it can go before. */
export function getMoveMenu(source: MoveMenuSource, words: MoveMenuWords, moved: string): MoveMenu {
  const targets = new Map<string, MoveTarget>()
  const children = source.listSlots().map((slot): MenuItem => {
    const before = source
      .listEntries(slot)
      .filter((entry) => entry.key !== moved)
      .map((entry): MenuItem => {
        const id = `before:${slot}:${entry.key}`
        targets.set(id, { slot, before: entry.key })
        return { id, text: entry.title }
      })
    const endId = `end:${slot}`
    targets.set(endId, { slot })
    return {
      id: `slot:${slot}`,
      text: source.getSlotTitle(slot),
      children: [...before, { id: endId, text: words.atTheEnd }],
    }
  })
  return { items: [{ id: 'move', text: words.moveTo, children }], targets }
}

/** What a move of the entry `key`, drawn in `from`, writes; empty when the move changes nothing. */
export function getMoveChange(
  layout: SlotLayout,
  listKeys: (slot: SlotName) => readonly string[],
  move: { readonly key: string; readonly from: SlotName; readonly target: MoveTarget },
): SlotLayoutChange {
  const { key, from, target } = move
  return moveSlotEntry(
    layout,
    { [from]: listKeys(from), [target.slot]: listKeys(target.slot) },
    { key, from, to: target.slot, before: target.before },
  )
}
