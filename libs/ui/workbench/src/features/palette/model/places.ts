/**
 * What the words turned up, in the order it is drawn.
 *
 * Nothing here draws anything.
 */
import { computed, type ComputedRef } from 'vue'
import { flatten, orderGroups, type PaletteGroup, type PalettePlace } from '../lib/item'
import { placePalette, type PlacedGroup } from '../lib/place'

export interface PalettePlacesState {
  readonly placed: ComputedRef<readonly PlacedGroup[]>
  readonly places: ComputedRef<readonly PalettePlace[]>
  readonly announcement: ComputedRef<string>
}

export interface PalettePlacesOptions {
  readonly groups: () => readonly PaletteGroup[]
}

export function usePalettePlaces(options: PalettePlacesOptions): PalettePlacesState {
  const visibleGroups = computed(() => orderGroups(options.groups()))

  const places = computed(() => flatten(visibleGroups.value))
  const groupsInOrder = computed(() => placePalette(visibleGroups.value))

  /**
   * What is announced from a search. A group with rows is announced by the row the
   * keyboard lands on; a group with none has no row to land on, so its empty-state
   * message is announced instead. A group still loading has no result yet and is
   * skipped.
   */
  const announcement = computed(() =>
    visibleGroups.value
      .filter((group) => !group.isWorking && group.items.length === 0 && group.silence)
      .map((group) => group.silence)
      .join('. '),
  )

  return { placed: groupsInOrder, places, announcement }
}
