/**
 * The user's arrangement of the bar slots, kept in the settings at `app.layout.slots`: the key of
 * an entry, the layout a settings value holds, and the lists a drop of an entry writes.
 */
import { BAR_SLOTS, type BarSlotName } from '@numen/plugin-sdk'
import type { DeclaredSlotEntry } from './slotRegistry'

/** The user's arrangement of the bar slots: per bar slot, the keys of its entries in drawing order. */
export type SlotLayout = Readonly<Partial<Record<BarSlotName, readonly string[]>>>

/** The lists of the bar slots a drop changes, each the whole new list of its slot. */
export type SlotLayoutChange = Readonly<Partial<Record<BarSlotName, readonly string[]>>>

/** One drop: the entry, the slot it is drawn in, the slot it is dropped in, and the key it is dropped before. */
export interface SlotMove {
  readonly key: string
  readonly from: BarSlotName
  readonly to: BarSlotName
  readonly before?: string | undefined
}

/** Whether a slot name is one of the bar slots. */
export const isBarSlot = (slot: string): slot is BarSlotName =>
  (BAR_SLOTS as readonly string[]).includes(slot)

/** The key of an entry in the layout, `<plugin id>/<view or command id>`; undefined for a submenu. */
export function getLayoutKey(entry: DeclaredSlotEntry): string | undefined {
  const id = entry.command ?? entry.view
  return id ? `${entry.pluginId}/${id}` : undefined
}

const isKeyList = (value: unknown): value is readonly string[] =>
  Array.isArray(value) && value.every((one) => typeof one === 'string')

/** The layout a settings value holds. A slot that is not a bar slot and a list that is not a list of strings are left out. */
export function readSlotLayout(value: unknown): SlotLayout {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return {}
  const layout: Partial<Record<BarSlotName, readonly string[]>> = {}
  for (const [slot, keys] of Object.entries(value)) {
    if (isBarSlot(slot) && isKeyList(keys)) layout[slot] = [...keys]
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
  drawn: readonly string[],
  next: readonly string[],
  moved: string,
): string[] {
  const list = [...next]
  let front = 0
  stored.forEach((key, index) => {
    if (key === moved || drawn.includes(key) || list.includes(key)) return
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

/** The lists a drop writes; empty when the drop leaves every slot as it is drawn. */
export function moveSlotEntry(
  layout: SlotLayout,
  drawn: Readonly<Record<BarSlotName, readonly string[]>>,
  move: SlotMove,
): SlotLayoutChange {
  const { key, from, to, before } = move
  const removed = drawn[from].filter((one) => one !== key)
  const inserted = insertBefore(from === to ? removed : drawn[to], key, before)
  if (from === to && isSameList(inserted, drawn[from])) return {}

  const orders: Partial<Record<BarSlotName, readonly string[]>> =
    from === to ? { [to]: inserted } : { [from]: removed, [to]: inserted }
  const change: Partial<Record<BarSlotName, readonly string[]>> = {}
  for (const slot of BAR_SLOTS) {
    const next = orders[slot]
    if (next === undefined) continue
    const list = mergeStoredKeys(layout[slot] ?? [], drawn[slot], next, key)
    if (!isSameList(list, layout[slot])) change[slot] = list
  }
  return change
}
