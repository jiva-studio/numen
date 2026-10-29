/**
 * What the words turned up, in the order it is drawn, and which of it the
 * keyboard is standing on.
 *
 * Nothing here draws anything.
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { stepTo } from '@numen/ui'
import {
  canChoose,
  canLandOn,
  flatten,
  orderGroups,
  type PaletteAction,
  type PaletteGroup,
  type PaletteItem,
  type PalettePlace,
} from '../lib/item'
import { getActionId } from '../lib/keys'
import { placePalette, type PlacedGroup } from '../lib/place'

export interface PalettePlacesState {
  /** The groups as they are drawn, their rows numbered over the whole list. */
  readonly placed: ComputedRef<readonly PlacedGroup[]>
  /** Every row of every group, in the order the keyboard walks them. */
  readonly places: ComputedRef<readonly PalettePlace[]>
  /** What a search came back with, where it came back with nothing. */
  readonly announcement: ComputedRef<string>
  /** The number of the row the keyboard is on, and -1 for none. */
  readonly here: ComputedRef<number>
  /** The item with keyboard focus, and the actions available on it. */
  readonly activeItem: ComputedRef<PaletteItem | undefined>
  readonly actions: ComputedRef<readonly PaletteAction[]>
  /** The keyboard onto the row at that number. */
  readonly goTo: (pos: number) => void
  /** The keyboard put on nothing, which a fresh question leaves it on. */
  readonly selectedId: Ref<string>
  /** A pointer crossing the list lights the row it passes over. */
  readonly lightAt: (pos: number, event: PointerEvent) => void
  /** The row at that index, and whether the second action was requested. */
  readonly chooseAt: (pos: number, second: boolean) => void
}

export interface PalettePlacesOptions {
  /** The groups, in the order they are offered. */
  readonly groups: () => readonly PaletteGroup[]
  /** What is typed, which a fresh question is a fresh list of. */
  readonly typed: Ref<string>
  /** The keyboard position, emitted whenever it moves. */
  readonly onItemActive: (item: string) => void
  /** An item chosen, and the action requested on it. */
  readonly tellChoice: (item: string, action: string) => void
}

export function usePalettePlaces(options: PalettePlacesOptions): PalettePlacesState {
  const { groups, typed } = options
  const visibleGroups = computed(() => orderGroups(groups()))

  const places = computed(() => flatten(visibleGroups.value))
  const placedPalette = computed(() => placePalette(visibleGroups.value))

  /**
   * What is announced from a search. A group with rows is announced by the row the
   * keyboard lands on; a group with none has no row to land on, so its empty-state
   * message is announced instead. A group still loading has no result yet and is
   * skipped.
   */
  const response = computed(() =>
    visibleGroups.value
      .filter((group) => !group.isWorking && group.items.length === 0 && group.silence)
      .map((group) => group.silence)
      .join('. '),
  )

  /** The item the keyboard is on, by its identity. */
  const target = ref('')

  const here = computed(() => places.value.findIndex((place) => place.item.id === target.value))

  const activeItem = computed(() => places.value[here.value]?.item)
  const proposed = computed(() => activeItem.value?.actions ?? [])

  const goTo = (pos: number): void => {
    target.value = places.value[pos]?.item.id ?? ''
  }

  /**
   * Answers arriving never move what is lit and never move the list. What they do
   * decide is the first item, when nothing is lit: a question was typed, and this
   * is its first answer.
   */
  watch(
    places,
    (now) => {
      if (here.value >= 0) return
      goTo(stepTo(now, { from: -1, by: 1 }, canLandOn))
    },
    { flush: 'post' },
  )

  /** A fresh question is a fresh list, and nothing in it is lit until it fills. */
  watch(typed, () => {
    target.value = ''
  })

  /**
   * The keyboard position, for a caller that displays the focused item. A moved
   * pointer highlights an item, so this is also emitted for a pointer crossing
   * the list.
   */
  watch(target, (now) => options.onItemActive(now), { flush: 'post' })

  /**
   * The previous pointer position. Only a pointer that has moved highlights an
   * item: a list scrolling under a stationary pointer reports a move too.
   */
  let status = { x: -1, y: -1 }

  const lightAt = (pos: number, event: PointerEvent): void => {
    if (event.clientX === status.x && event.clientY === status.y) return
    status = { x: event.clientX, y: event.clientY }
    // An item the keyboard steps over is one the pointer passes over.
    const item = places.value[pos]?.item
    if (item && canChoose(item)) goTo(pos)
  }

  const chooseAt = (pos: number, second: boolean): void => {
    const item = places.value[pos]?.item
    const action = getActionId(item, second)
    if (!item || !action) return
    options.tellChoice(item.id, action)
  }

  const state = {
    placed: placedPalette,
    places,
    announcement: response,
    here,
    activeItem,
    actions: proposed,
    goTo,
    selectedId: target,
    lightAt,
    chooseAt,
  }
  return state
}
