/**
 * The user's arrangement of the slots, kept in the settings at `app.layout.slots`: the key of an
 * entry, the layout a settings value holds, and the lists a move of an entry writes. Only the slots
 * whose arrangement is `user` in `slots.json` take a list.
 */
import { isUserArrangedSlot, listUserArrangedSlots, type SlotName } from '@numen/plugin-sdk'
import type { DeclaredSlotEntry } from './slotRegistry'

/** The user's arrangement of the slots: per slot, the keys of its entries in drawing order. */
export type SlotLayout = Readonly<Partial<Record<SlotName, readonly string[]>>>

/** The lists of the slots a move changes, each the whole new list of its slot. */
export type SlotLayoutChange = Readonly<Partial<Record<SlotName, readonly string[]>>>

/** One move: the entry, the slot it is drawn in, the slot it goes to, and the key it goes before. */
export interface SlotMove {
  readonly key: string
  readonly from: SlotName
  readonly to: SlotName
  readonly before?: string | undefined
}

/** The key of an entry in the layout, `<home slot>|<plugin id>/<view or command id>`; undefined for a submenu. */
export function getLayoutKey(entry: DeclaredSlotEntry): string | undefined {
  const id = entry.command ?? entry.view
  return id ? `${entry.slot}|${entry.pluginId}/${id}` : undefined
}

/** Where a key of the layout is: the first slot, in `slots.json` order, that lists it, and its first index there. */
export interface LayoutPlace {
  readonly slot: SlotName
  readonly index: number
}

/** The place of each key in the layout. */
export function findLayoutPlaces(layout: SlotLayout): ReadonlyMap<string, LayoutPlace> {
  const places = new Map<string, LayoutPlace>()
  for (const slot of listUserArrangedSlots()) {
    ;(layout[slot] ?? []).forEach((key, index) => {
      if (!places.has(key)) places.set(key, { slot, index })
    })
  }
  return places
}

const isKeyList = (value: unknown): value is readonly string[] =>
  Array.isArray(value) && value.every((one) => typeof one === 'string')

/** The layout a settings value holds. A slot that is not arranged by the user and a list that is not a list of strings are left out. */
export function readSlotLayout(value: unknown): SlotLayout {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return {}
  const layout: Partial<Record<SlotName, readonly string[]>> = {}
  for (const [slot, keys] of Object.entries(value)) {
    if (isUserArrangedSlot(slot) && isKeyList(keys)) layout[slot] = [...keys]
  }
  return layout
}

const isSameList = (a: readonly string[], b: readonly string[] | undefined): boolean =>
  b !== undefined && a.length === b.length && a.every((key, index) => key === b[index])

const insertBefore = (
  keys: readonly string[],
  key: string,
  before: string | undefined,
): string[] => {
  const index = before === undefined ? -1 : keys.indexOf(before)
  return index < 0 ? [...keys, key] : [...keys.slice(0, index), key, ...keys.slice(index)]
}

/**
 * The new list of a slot: its new drawn order, with each stored key that is not drawn put back
 * right after the stored key that preceded it, or first when none did.
 */
function mergeStoredKeys(
  stored: readonly string[],
  next: readonly string[],
  isDrawn: (key: string) => boolean,
): string[] {
  const list = [...next]
  let front = 0
  stored.forEach((key, index) => {
    if (isDrawn(key) || list.includes(key)) return
    const previous = stored
      .slice(0, index)
      .reverse()
      .find((one) => list.includes(one))
    if (previous === undefined) {
      list.splice(front, 0, key)
      front += 1
    } else {
      list.splice(list.indexOf(previous) + 1, 0, key)
    }
  })
  return list
}

type DrawnKeys = Readonly<Partial<Record<SlotName, readonly string[]>>>

const getDrawn = (drawn: DrawnKeys, slot: SlotName): readonly string[] => drawn[slot] ?? []

/** The lists a move writes; empty when the move leaves every slot as it is drawn. */
export function moveSlotEntry(
  layout: SlotLayout,
  drawn: DrawnKeys,
  move: SlotMove,
): SlotLayoutChange {
  const { key, from, to, before } = move
  const removed = getDrawn(drawn, from).filter((one) => one !== key)
  const inserted = insertBefore(from === to ? removed : getDrawn(drawn, to), key, before)
  if (from === to && isSameList(inserted, getDrawn(drawn, from))) return {}

  const orders: (readonly [SlotName, readonly string[]])[] =
    from === to
      ? [[to, inserted]]
      : [
          [from, removed],
          [to, inserted],
        ]
  const change: Partial<Record<SlotName, readonly string[]>> = {}
  for (const [slot, next] of orders) {
    const list = mergeStoredKeys(
      layout[slot] ?? [],
      next,
      (one) => one === key || getDrawn(drawn, slot).includes(one),
    )
    if (!isSameList(list, layout[slot])) change[slot] = list
  }
  return change
}
