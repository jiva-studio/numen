/**
 * What a palette is, as plain values. No DOM, no measurement, no clock.
 */
import type { Span } from '@/shared/lib/span'
import type { PaletteKeys } from '@/shared/ui/key-cap'
import type { PaletteAction } from '../types'

export type { PaletteAction }

/**
 * One thing that can be chosen. The identifier is opaque: the palette has no
 * way to ask what it addresses, and hands it back as given.
 */
export interface PaletteItem {
  readonly id: string
  /** What it is called. */
  readonly title: string
  /** Where in the title the words stand. */
  readonly at?: readonly Span[]
  /** A second line: where the item stands, or the words it was found among. */
  readonly detail?: string
  /** Where the words stand in that line. */
  readonly detailAt?: readonly Span[]
  /** What can be done to it. An item offering none is drawn and not chosen. */
  readonly actions?: readonly PaletteAction[]
  /**
   * The keystroke that reaches this item away from the palette, as the caller
   * holds it. Drawn at the end of the row.
   */
  readonly keys?: PaletteKeys
  /** Drawn and announced, and not choosable. */
  readonly disabled?: boolean
}

/** One group of the list. What the groups are is the caller's. */
export interface PaletteGroup {
  readonly id: string
  /** What the group is called. */
  readonly title: string
  readonly items: readonly PaletteItem[]
  /** More is on its way, so what stands here is not all of it. */
  readonly isWorking?: boolean
  /**
   * What is said in place of items when the group holds none. A group with
   * nothing to say here and nothing on its way is drawn nowhere.
   */
  readonly silence?: string
}

/**
 * The item the keyboard is standing on, by the identity it was given, and the
 * empty string where it stands on none.
 */
export type PaletteLit = string

/** One item, and the group it was drawn in. */
export interface PalettePlace {
  readonly group: PaletteGroup
  readonly item: PaletteItem
}

/**
 * The groups in the order they are drawn: as they were offered, and the ones
 * holding nothing after the ones holding something.
 *
 * A group holding nothing is drawn while it is still working, and where it has
 * something to say in place of items. One that is neither is worth no heading
 * of its own and is drawn nowhere. Such a group holds no item either way, so
 * what the keyboard counts is untouched.
 */
export const orderGroups = (groups: readonly PaletteGroup[]): readonly PaletteGroup[] => [
  ...groups.filter((one) => one.items.length > 0),
  ...groups.filter((one) => one.items.length === 0 && (one.isWorking || Boolean(one.silence))),
]

/**
 * Every item in the order it is drawn, so that one number says which item.
 */
export const flatten = (groups: readonly PaletteGroup[]): readonly PalettePlace[] =>
  groups.flatMap((group) => group.items.map((item) => ({ group, item })))

/** Whether the keyboard may land here. An item with nothing to do is passed over. */
export const choosable = (item: PaletteItem): boolean =>
  !item.disabled && (item.actions?.length ?? 0) > 0

/**
 * Where the keyboard lands next, counting from `from` and passing over what
 * cannot be chosen. It wraps, and answers -1 when there is nothing to land on.
 *
 * Counting from -1 by one is how the first is asked for, and from 0 by minus
 * one is how the last is.
 */
export const stepTo = (places: readonly PalettePlace[], from: number, by: number): number => {
  const total = places.length
  for (let step = 1; step <= total; step += 1) {
    const at = (((from + by * step) % total) + total) % total
    const place = places[at]
    if (place && choosable(place.item)) return at
  }
  return -1
}

/**
 * Where the keyboard lands in a list of this many, counting from `from`. It
 * wraps, and answers -1 for a list holding nothing. Every row of such a list
 * can be landed on.
 */
export const stepIn = (total: number, from: number, by: number): number => {
  if (total <= 0) return -1
  return (((from + by) % total) + total) % total
}

/**
 * Where the keyboard stands once the list has changed under it: on the item it
 * was on, wherever that item has moved to. An item that is gone hands it to the
 * first item there is; a list with nothing to land on takes it nowhere.
 */
export const findKeptPlace = (places: readonly PalettePlace[], was: string): number => {
  const held = places.findIndex((place) => place.item.id === was && choosable(place.item))
  return held >= 0 ? held : stepTo(places, -1, 1)
}
