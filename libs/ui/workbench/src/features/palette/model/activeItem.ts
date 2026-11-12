/**
 * The item the keyboard is standing on: which place it holds, how it moves, and what choosing
 * does.
 *
 * Nothing here draws anything.
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { stepTo } from '@numen/ui'
import {
  canChoose,
  canLandOn,
  type PaletteAction,
  type PaletteItem,
  type PalettePlace,
} from '../lib/item'
import { getActionId } from '../lib/keys'

export interface ActiveItemState {
  readonly here: ComputedRef<number>
  readonly activeItem: ComputedRef<PaletteItem | undefined>
  readonly actions: ComputedRef<readonly PaletteAction[]>
  readonly goTo: (pos: number) => void
  readonly selectedId: Ref<string>
  readonly lightAt: (pos: number, event: PointerEvent) => void
  readonly chooseAt: (pos: number, second: boolean) => void
}

export interface ActiveItemOptions {
  readonly places: ComputedRef<readonly PalettePlace[]>
  readonly typed: Ref<string>
  readonly onItemActive: (item: string) => void
  readonly tellChoice: (item: string, action: string) => void
}

export function useActiveItem(options: ActiveItemOptions): ActiveItemState {
  const { places, typed } = options

  /** The item the keyboard is on, by its identity. */
  const target = ref('')

  const here = computed(() => places.value.findIndex((place) => place.item.id === target.value))

  const activeItem = computed(() => places.value[here.value]?.item)
  const actions = computed(() => activeItem.value?.actions ?? [])

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

  return { here, activeItem, actions, goTo, selectedId: target, lightAt, chooseAt }
}
