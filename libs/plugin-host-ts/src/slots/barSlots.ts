/**
 * The bar slots as the user arranges them: the entries the layout lists for a slot, in its order,
 * then the entries placed there that no bar slot lists.
 */
import { computed } from 'vue'
import { BAR_SLOTS, type BarSlotName, type CommandTarget, type SlotItem } from '@numen/plugin-sdk'
import { getLayoutKey, isBarSlot, type SlotLayout } from './slotLayout'
import { compareInSlot, keepFirstOfEachCommand, type Placed } from './slotOrder'
import type { DeclaredSlotEntry } from './slotRegistry'

/**
 * The items of a bar slot: the entries its layout lists, in the order of their first place in the
 * list, then the entries placed there that no bar slot lists, by group, order and plugin id. A
 * separator is drawn between two neighbours of different groups.
 */
function layOutBar(
  slot: BarSlotName,
  placed: readonly Placed[],
  getLayoutIndex: (entry: DeclaredSlotEntry) => number | undefined,
): SlotItem[] {
  const inLayout: { readonly one: Placed; readonly index: number }[] = []
  const outsideLayout: Placed[] = []
  for (const one of keepFirstOfEachCommand(placed)) {
    const index = getLayoutIndex(one.entry)
    if (index === undefined) outsideLayout.push(one)
    else inLayout.push({ one, index })
  }
  const ordered = [
    ...inLayout
      .sort((a, b) => a.index - b.index || compareInSlot(a.one, b.one))
      .map(({ one }) => one),
    ...outsideLayout.sort(compareInSlot),
  ]
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

/** Where a key of the layout is: the first bar slot that lists it, and its index in that list. */
interface LayoutPlace {
  readonly slot: BarSlotName
  readonly index: number
}

/** The place of each key in the layout, by the first bar slot that lists it and its first index there. */
function findLayoutPlaces(layout: SlotLayout): ReadonlyMap<string, LayoutPlace> {
  const places = new Map<string, LayoutPlace>()
  for (const slot of BAR_SLOTS) {
    ;(layout[slot] ?? []).forEach((key, index) => {
      if (!places.has(key)) places.set(key, { slot, index })
    })
  }
  return places
}

export interface BarSlotsDeps {
  readonly getEntries: () => readonly DeclaredSlotEntry[]
  readonly getLayout: () => SlotLayout | undefined
  readonly getItemKey: (entry: DeclaredSlotEntry) => string | undefined
  readonly toItem: (
    entry: DeclaredSlotEntry,
    target: CommandTarget | undefined,
  ) => SlotItem | undefined
}

/** The items of the bar slots, and the layout key of each item an entry of a bar slot draws. */
export function createBarSlots(deps: BarSlotsDeps) {
  const layoutPlaces = computed(() => findLayoutPlaces(deps.getLayout() ?? {}))

  /** The layout key of each item key an entry of a bar slot is drawn under. */
  const layoutKeys = computed(() => {
    const keys = new Map<string, string>()
    for (const entry of deps.getEntries()) {
      const itemKey = deps.getItemKey(entry)
      const layoutKey = getLayoutKey(entry)
      if (isBarSlot(entry.slot) && itemKey && layoutKey) keys.set(itemKey, layoutKey)
    }
    return keys
  })

  const resolveBar = (place: BarSlotName, target: CommandTarget | undefined): SlotItem[] => {
    const places = layoutPlaces.value
    const findPlace = (entry: DeclaredSlotEntry) => {
      const key = getLayoutKey(entry)
      return key === undefined ? undefined : places.get(key)
    }
    const items: Placed[] = []
    for (const entry of deps.getEntries()) {
      if (!isBarSlot(entry.slot)) continue
      if ((findPlace(entry)?.slot ?? entry.slot) !== place) continue
      const item = deps.toItem(entry, target)
      if (item) items.push({ entry, item })
    }
    return layOutBar(place, items, (entry) => findPlace(entry)?.index)
  }

  return {
    resolveBar,
    findLayoutKey: (itemKey: string): string | undefined => layoutKeys.value.get(itemKey),
  }
}
