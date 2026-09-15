/** The list as it is drawn: every row numbered, and its lines already split. */
import type { PaletteGroup, PaletteItem } from './item'
import { partsOf, type PalettePart } from './parts'

/** One item as it is drawn: the number it has in the list, and its lines split. */
export interface PlacedItem {
  readonly item: PaletteItem
  /** Its number in the whole list, which is what the keyboard counts in. */
  readonly at: number
  readonly name: readonly PalettePart[]
  readonly detail: readonly PalettePart[]
}

/** One group as it is drawn. */
export interface PlacedGroup {
  readonly group: PaletteGroup
  readonly items: readonly PlacedItem[]
}

/**
 * Every group, with its items numbered as they stand in the whole list and each
 * of their lines already split into runs.
 *
 * The groups are walked in the order they were given, which is the order the
 * keyboard counts in, so a number here is a number into `flatten`.
 */
export const placePalette = (groups: readonly PaletteGroup[]): readonly PlacedGroup[] => {
  let at = 0
  return groups.map((group) => ({
    group,
    items: group.items.map((item) => ({
      item,
      at: at++,
      name: partsOf(item.title, item.at),
      detail: partsOf(item.detail ?? '', item.detailAt),
    })),
  }))
}

/** What the list of answers is addressed by, so the field can point at it. */
export const listId = (uid: string): string => `${uid}-list`

/** What one row is addressed by, so the field can name the row that is lit. */
export const optionId = (uid: string, at: number): string => `${uid}-option-${at}`
