/** The room each node keeps over and under its pill for the widgets of plugins, as their measured height, and whether their views have loaded. */
import { computed, onScopeDispose, shallowRef, type ComputedRef, type Ref } from 'vue'
import { measureRect } from '@numen/ui'
import { NO_ROOM, type NodeRoom, type PlexOptions } from '../lib/arrange'
import type { PlexNode } from '../../types'

export type WidgetSide = 'above' | 'below'

interface Measured {
  readonly element: Element
  readonly observer: ResizeObserver
}

/**
 * The height assumed over or under a node for the widgets of a side that are not measured yet, about two rows of them and the gap to the node,
 * so the notes chosen before any widget is drawn are close to the notes chosen once they are.
 */
export const RESERVED_WIDGETS_HEIGHT = 88

const getKey = (id: string, side: WidgetSide): string => `${side}:${id}`

/**
 * The widgets of each side of a node are measured whenever `setWidgets` is given their element, and again when it resizes;
 * `null` takes them away. `getDraftRoom` is the same with `RESERVED_WIDGETS_HEIGHT` for each side with widgets that is not measured. `getNodeRoom` is what the arrangement is given: a new function whenever a measured height
 * changes, and absent while nothing is measured. `setReady` takes whether the views of a side have loaded and
 * `null` when the side is removed; `areLoaded` holds when every side of every given node has reported.
 */
export function useWidgetRooms(
  options: Readonly<Ref<PlexOptions>>,
  hasWidgets: (side: WidgetSide) => boolean = () => false,
): {
  setWidgets: (id: string, side: WidgetSide, element: Element | null) => void
  getNodeRoom: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
  getDraftRoom: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
  setReady: (id: string, side: WidgetSide, isReady: boolean | null) => void
  areLoaded: (ids: readonly string[]) => boolean
} {
  const heights = shallowRef<ReadonlyMap<string, number>>(new Map())
  const measured = new Map<string, Measured>()
  const loaded = shallowRef<ReadonlyMap<string, boolean>>(new Map())

  const setHeight = (key: string, height: number | undefined) => {
    if (heights.value.get(key) === height) return
    const next = new Map(heights.value)
    if (height === undefined) next.delete(key)
    else next.set(key, height)
    heights.value = next
  }

  const release = (key: string) => {
    measured.get(key)?.observer.disconnect()
    measured.delete(key)
    setHeight(key, undefined)
  }

  const setWidgets = (id: string, side: WidgetSide, element: Element | null) => {
    const key = getKey(id, side)
    if (element && measured.get(key)?.element === element) {
      setHeight(key, measureRect(element).height)
      return
    }
    release(key)
    if (!element) return
    const observer = new ResizeObserver(() => setHeight(key, measureRect(element).height))
    measured.set(key, { element, observer })
    observer.observe(element)
    setHeight(key, measureRect(element).height)
  }

  const setReady = (id: string, side: WidgetSide, isReady: boolean | null) => {
    const key = getKey(id, side)
    if ((loaded.value.get(key) ?? null) === isReady) return
    const next = new Map(loaded.value)
    if (isReady === null) next.delete(key)
    else next.set(key, isReady)
    loaded.value = next
  }

  const areLoaded = (ids: readonly string[]) =>
    ids.every(
      (id) => loaded.value.get(getKey(id, 'above')) && loaded.value.get(getKey(id, 'below')),
    )

  onScopeDispose(() => {
    for (const { observer } of measured.values()) observer.disconnect()
    measured.clear()
  })

  const createRoom = (isDraft: boolean) => {
    const found = heights.value
    const hasAbove = isDraft && hasWidgets('above')
    const hasBelow = isDraft && hasWidgets('below')
    if (found.size === 0 && !hasAbove && !hasBelow) return undefined
    const { width } = options.value.nodeSize
    return (node: PlexNode): NodeRoom => {
      const top =
        found.get(getKey(node.id, 'above')) ?? (hasAbove ? RESERVED_WIDGETS_HEIGHT : undefined)
      const bottom =
        found.get(getKey(node.id, 'below')) ?? (hasBelow ? RESERVED_WIDGETS_HEIGHT : undefined)
      return top === undefined && bottom === undefined
        ? NO_ROOM
        : { top: top ?? 0, bottom: bottom ?? 0, width }
    }
  }
  const getNodeRoom = computed(() => createRoom(false))
  const getDraftRoom = computed(() => createRoom(true))

  return { setWidgets, getNodeRoom, getDraftRoom, setReady, areLoaded }
}
