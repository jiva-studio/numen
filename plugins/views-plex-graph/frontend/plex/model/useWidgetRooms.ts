/** The room each node keeps over and under its pill for the widgets of plugins, as their measured height, and whether their views have loaded. */
import { computed, onScopeDispose, shallowRef, type ComputedRef, type Ref } from 'vue'
import { measureRect } from '@numen/ui'
import { NO_ROOM, type NodeRoom, type PlexOptions } from '../lib/arrange'
import type { PlexNode, PlexWidget } from '../../types'

export type WidgetSide = 'above' | 'below'

interface Measured {
  readonly element: Element
  readonly observer: ResizeObserver
}

/** The views a plugin places over and under the nodes. */
interface PlexWidgetLists {
  readonly widgetsAbove?: readonly PlexWidget[] | undefined
  readonly widgetsBelow?: readonly PlexWidget[] | undefined
}

const getKey = (id: string, side: WidgetSide): string => `${side}:${id}`

/**
 * The widgets of each side of a node are measured whenever `setWidgets` is given their element, and again when it resizes;
 * `null` takes them away. `getNodeRoom` is what the arrangement is given: a new function whenever a measured height
 * changes, and absent while nothing is measured. `setReady` takes whether the views of a side have loaded and
 * `null` when the side is removed; `areLoaded` holds when every side of every given node has reported.
 */
export function useWidgetRooms(
  options: Readonly<Ref<PlexOptions>>,
  widgets: PlexWidgetLists = {},
): {
  hasWidgets: (side?: WidgetSide) => boolean
  setWidgets: (id: string, side: WidgetSide, element: Element | null) => void
  getNodeRoom: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
  setReady: (id: string, side: WidgetSide, isReady: boolean | null) => void
  areLoaded: (ids: readonly string[]) => boolean
} {
  const hasWidgets = (side?: WidgetSide) =>
    !!(
      (side !== 'below' && widgets.widgetsAbove?.length) ||
      (side !== 'above' && widgets.widgetsBelow?.length)
    )
  const heights = shallowRef<ReadonlyMap<string, number>>(new Map())
  const measured = new Map<string, Measured>()
  const loaded = shallowRef<ReadonlyMap<string, boolean>>(new Map())

  const setHeight = (key: string, height: number | undefined) => {
    if (heights.value.get(key) === (height || undefined)) return
    const next = new Map(heights.value)
    if (!height) next.delete(key)
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

  // Without widgets there is nothing to wait for.
  const areLoaded = (ids: readonly string[]) =>
    !hasWidgets() ||
    ids.every(
      (id) => loaded.value.get(getKey(id, 'above')) && loaded.value.get(getKey(id, 'below')),
    )

  onScopeDispose(() => {
    for (const { observer } of measured.values()) observer.disconnect()
    measured.clear()
  })

  const getNodeRoom = computed(() => {
    const found = heights.value
    if (found.size === 0) return undefined
    const { width } = options.value.nodeSize
    return (node: PlexNode): NodeRoom => {
      const top = found.get(getKey(node.id, 'above'))
      const bottom = found.get(getKey(node.id, 'below'))
      return top === undefined && bottom === undefined
        ? NO_ROOM
        : { top: top ?? 0, bottom: bottom ?? 0, width }
    }
  })

  return { hasWidgets, setWidgets, getNodeRoom, setReady, areLoaded }
}
