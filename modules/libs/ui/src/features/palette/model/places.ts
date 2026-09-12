/**
 * What the words turned up, in the order it is drawn, and which of it the
 * keyboard is standing on.
 *
 * A group holding nothing stands at the foot, whatever order it was offered in.
 * Nothing here draws anything.
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import {
  choosable,
  flatten,
  orderGroups,
  stepTo,
  type PaletteAction,
  type PaletteGroup,
  type PaletteItem,
  type PalettePlace,
} from '../lib/item'
import { actionAt } from '../lib/keys'
import { placePalette, type PlacedGroup } from '../lib/place'

export interface PalettePlacesState {
  /** The groups as they are drawn, their rows numbered over the whole list. */
  readonly placed: ComputedRef<readonly PlacedGroup[]>
  /** Every row of every group, in the order the keyboard walks them. */
  readonly places: ComputedRef<readonly PalettePlace[]>
  /** What a search came back with, where it came back with nothing. */
  readonly said: ComputedRef<string>
  /** The number of the row the keyboard is on, and -1 for none. */
  readonly here: ComputedRef<number>
  /** The item the keyboard is on, and what it can be asked. */
  readonly lit: ComputedRef<PaletteItem | undefined>
  readonly offered: ComputedRef<readonly PaletteAction[]>
  /** The keyboard onto the row at that number. */
  readonly goTo: (at: number) => void
  /** The keyboard put on nothing, which a fresh question leaves it on. */
  readonly held: Ref<string>
  /** A pointer crossing the list lights the row it passes over. */
  readonly lightAt: (at: number, event: PointerEvent) => void
  /** The row at that number chosen, and whether the second action was asked for. */
  readonly chooseAt: (at: number, second: boolean) => void
}

export interface PalettePlacesOptions {
  /** The groups, in the order they are offered. */
  readonly groups: () => readonly PaletteGroup[]
  /** What is typed, which a fresh question is a fresh list of. */
  readonly typed: Ref<string>
  /** Where the keyboard is standing, said whenever it moves. */
  readonly tellLit: (item: string) => void
  /** An item chosen, and what was asked of it. */
  readonly tellChoice: (item: string, action: string) => void
}

export function usePalettePlaces(options: PalettePlacesOptions): PalettePlacesState {
  const { groups, typed } = options
  /** What is drawn, and in what order: a group holding nothing stands at the foot. */
  const shown = computed(() => orderGroups(groups()))

  const places = computed(() => flatten(shown.value))
  const placed = computed(() => placePalette(shown.value))

  /**
   * What is read out of a search. A group that answered with rows is read out by
   * the row the keyboard lands on; a group that answered with none has no row to
   * land on, so what it says in place of one is read out instead. A group still
   * working has answered nothing yet and is left alone.
   */
  const said = computed(() =>
    shown.value
      .filter((group) => !group.working && group.items.length === 0 && group.silence)
      .map((group) => group.silence)
      .join('. '),
  )

  /** The item the keyboard is on, by its identity. */
  const held = ref('')

  const here = computed(() => places.value.findIndex((place) => place.item.id === held.value))

  const lit = computed(() => places.value[here.value]?.item)
  const offered = computed(() => lit.value?.actions ?? [])

  const goTo = (at: number): void => {
    held.value = places.value[at]?.item.id ?? ''
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
      goTo(stepTo(now, -1, 1))
    },
    { flush: 'post' },
  )

  /** A fresh question is a fresh list, and nothing in it is lit until it fills. */
  watch(typed, () => {
    held.value = ''
  })

  /**
   * Where the keyboard is standing, for a caller that shows what it is standing
   * on. A pointer that has moved lights an item, so this is said for a pointer
   * crossing the list too.
   */
  watch(held, (now) => options.tellLit(now), { flush: 'post' })

  /**
   * Where the pointer was. Only a pointer that has actually moved lights what
   * it is over: a list scrolling under a pointer standing still reports a move
   * too.
   */
  let stood = { x: -1, y: -1 }

  const lightAt = (at: number, event: PointerEvent): void => {
    if (event.clientX === stood.x && event.clientY === stood.y) return
    stood = { x: event.clientX, y: event.clientY }
    // An item the keyboard steps over is one the pointer passes over.
    const item = places.value[at]?.item
    if (item && choosable(item)) goTo(at)
  }

  const chooseAt = (at: number, second: boolean): void => {
    const item = places.value[at]?.item
    const action = actionAt(item, second)
    if (!item || !action) return
    options.tellChoice(item.id, action)
  }

  return { placed, places, said, here, lit, offered, goTo, held, lightAt, chooseAt }
}
