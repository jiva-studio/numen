/**
 * The user's arrangement of the bar slots. The contract of
 * `.agents/tasks/1803-slot-layout/spec.md` §1.7, declared so its acceptance tests load; the bodies
 * are not written yet.
 */
import type { BarSlotName } from '@numen/plugin-sdk'
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

/** The key of an entry in the layout, `<plugin id>/<view or command id>`; undefined for a submenu. */
export function getLayoutKey(_entry: DeclaredSlotEntry): string | undefined {
  throw new Error('getLayoutKey is not implemented')
}

/** The layout a settings value holds. A slot that is not a bar slot and a list that is not a list of strings are left out. */
export function readSlotLayout(_value: unknown): SlotLayout {
  throw new Error('readSlotLayout is not implemented')
}

/** The lists a drop writes; empty when the drop leaves every slot as it is drawn. */
export function moveSlotEntry(
  _layout: SlotLayout,
  _drawn: Readonly<Record<BarSlotName, readonly string[]>>,
  _move: SlotMove,
): SlotLayoutChange {
  throw new Error('moveSlotEntry is not implemented')
}
