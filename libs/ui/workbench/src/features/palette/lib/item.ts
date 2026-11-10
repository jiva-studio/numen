/**
 * What a palette is, as plain values. No DOM, no measurement, no clock.
 */
import { stepTo } from '@numen/ui'
import type { TextRange } from '@numen/ui'
import type { Keystroke } from '@numen/ui'
import type { PaletteAction } from '../types'

export type { PaletteAction }

/**
 * One selectable entry. The identifier is opaque: the palette cannot resolve
 * what it addresses, and returns it unchanged.
 */
export interface PaletteItem {
  readonly id: string
  readonly title: string
  readonly at?: readonly TextRange[]
  readonly detail?: string
  readonly detailAt?: readonly TextRange[]
  readonly actions?: readonly PaletteAction[]
  readonly keys?: Keystroke
  readonly isDisabled?: boolean
}

/** One group of the list. What the groups are is the caller's. */
export interface PaletteGroup {
  readonly id: string
  readonly title: string
  readonly items: readonly PaletteItem[]
  readonly isWorking?: boolean
  readonly silence?: string
}

/**
 * The item the keyboard is standing on, by the identity it was given, and the
 * empty string where it relies on none.
 */
export type PaletteItemId = string

/** One item, and the group it was drawn in. */
export interface PalettePlace {
  readonly group: PaletteGroup
  readonly item: PaletteItem
}

/**
 * The groups holding something, then the groups holding nothing.
 * A group holding nothing is kept while it is still working or has something to say in place of items.
 * Any other empty group is dropped. Empty groups hold no item, so what the keyboard counts is untouched.
 */
export const orderGroups = (groups: readonly PaletteGroup[]): readonly PaletteGroup[] => [
  ...groups.filter((one) => one.items.length > 0),
  ...groups.filter((one) => one.items.length === 0 && (one.isWorking || Boolean(one.silence))),
]

/**
 * Every item in display order. A single index identifies an item.
 */
export const flatten = (groups: readonly PaletteGroup[]): readonly PalettePlace[] =>
  groups.flatMap((group) => group.items.map((item) => ({ group, item })))

/** Whether the keyboard may land here. An item with nothing to do is passed over. */
export const canChoose = (item: PaletteItem): boolean =>
  !item.isDisabled && (item.actions?.length ?? 0) > 0

/** Whether the keyboard may land on the place. */
export const canLandOn = (place: PalettePlace): boolean => canChoose(place.item)

/**
 * Where the keyboard lands in a list of this many, counting from `from`. It
 * wraps, and answers -1 for a list holding nothing. Every row of such a list
 * can be landed on.
 */
export const getSteppedIndex = (total: number, from: number, delta: number): number => {
  if (total <= 0) return -1
  return (((from + delta) % total) + total) % total
}

/**
 * Where the keyboard focus goes after the list changed: on the same item,
 * wherever it moved to. If the item is gone, focus goes to the first available
 * item; if the list has no selectable item, it goes to none.
 */
export const findKeptPlace = (places: readonly PalettePlace[], was: string): number => {
  const pos = places.findIndex((place) => place.item.id === was && canChoose(place.item))
  return stepTo(places, { from: Math.max(pos, 0) - 1, by: 1 }, canLandOn)
}
