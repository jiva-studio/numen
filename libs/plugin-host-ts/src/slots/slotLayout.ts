/**
 * The user's arrangement of the slots, kept in the settings at `app.layout.slots`: the key of an
 * entry, the layout a settings value holds, and the lists a drop of an entry writes.
 */
import { SLOT, type SlotName } from '@numen/plugin-sdk'
import type { DeclaredSlotEntry } from './slotRegistry'

/** The user's arrangement of the slots: per slot, the keys of its entries in drawing order. */
export type SlotLayout = Readonly<Partial<Record<SlotName, readonly string[]>>>

/** The lists of the slots a drop changes, each the whole new list of its slot. */
export type SlotLayoutChange = Readonly<Partial<Record<SlotName, readonly string[]>>>

/** One drop: the entry, the slot it is drawn in, the slot it is dropped in, and the key it is dropped before. */
export interface SlotMove {
  readonly key: string
  readonly from: SlotName
  readonly to: SlotName
  readonly before?: string | undefined
}

const SLOT_NAMES: readonly SlotName[] = Object.values(SLOT)

const isSlotName = (slot: string): slot is SlotName =>
  (SLOT_NAMES as readonly string[]).includes(slot)

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
  for (const slot of SLOT_NAMES) {
    ;(layout[slot] ?? []).forEach((key, index) => {
      if (!places.has(key)) places.set(key, { slot, index })
    })
  }
  return places
}

const isKeyList = (value: unknown): value is readonly string[] =>
  Array.isArray(value) && value.every((one) => typeof one === 'string')

/** The layout a settings value holds. A slot that \`slots.json\` does not declare and a list that is not a list of strings are left out. */
export function readSlotLayout(value: unknown): SlotLayout {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return {}
  const layout: Partial<Record<SlotName, readonly string[]>> = {}
  for (const [slot, keys] of Object.entries(value)) {
    if (isSlotName(slot) && isKeyList(keys)) layout[slot] = [...keys]
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

/** The lists a drop writes; empty when the drop leaves every slot as it is drawn. */
export function moveSlotEntry(
  layout: SlotLayout,
  drawn: DrawnKeys,
  move: SlotMove,
): SlotLayoutChange {
  const { key, from, to, before } = move
  const removed = getDrawn(drawn, from).filter((one) => one !== key)
  const inserted = insertBefore(from === to ? removed : getDrawn(drawn, to), key, before)
  if (from === to && isSameList(inserted, getDrawn(drawn, from))) return {}

  const orders: Partial<Record<SlotName, readonly string[]>> =
    from === to ? { [to]: inserted } : { [from]: removed, [to]: inserted }
  const change: Partial<Record<SlotName, readonly string[]>> = {}
  for (const slot of SLOT_NAMES) {
    const next = orders[slot]
    if (next === undefined) continue
    const list = mergeStoredKeys(
      layout[slot] ?? [],
      next,
      (one) => one === key || getDrawn(drawn, slot).includes(one),
    )
    if (!isSameList(list, layout[slot])) change[slot] = list
  }
  return change
}
