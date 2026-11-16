/** The room each node keeps over and under its pill for the widgets of plugins, as their measured height. */
import { computed, onScopeDispose, shallowRef, type ComputedRef, type Ref } from 'vue'
import { measureRect } from '@numen/ui'
import { NO_ROOM, type NodeRoom, type PlexOptions } from '../lib/arrange'
import type { PlexNode } from '../../types'

export type WidgetSide = 'above' | 'below'

interface Held {
  readonly element: Element
  readonly observer: ResizeObserver
}

const getKey = (id: string, side: WidgetSide): string => `${side}:${id}`

/**
 * The widgets of each side of a node are measured when `setWidgets` is given their element and again when it resizes;
 * `null` takes them away. `getNodeRoom` is what the arrangement is given: a new function whenever a measured height
 * changes, and absent while nothing is measured.
 */
export function useWidgetRooms(options: Readonly<Ref<PlexOptions>>): {
  setWidgets: (id: string, side: WidgetSide, element: Element | null) => void
  getNodeRoom: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
} {
  const heights = shallowRef<ReadonlyMap<string, number>>(new Map())
  const held = new Map<string, Held>()

  const setHeight = (key: string, height: number | undefined) => {
    if (heights.value.get(key) === height) return
    const next = new Map(heights.value)
    if (height === undefined) next.delete(key)
    else next.set(key, height)
    heights.value = next
  }

  const release = (key: string) => {
    held.get(key)?.observer.disconnect()
    held.delete(key)
    setHeight(key, undefined)
  }

  const setWidgets = (id: string, side: WidgetSide, element: Element | null) => {
    const key = getKey(id, side)
    if (held.get(key)?.element === element) return
    release(key)
    if (!element) return
    const observer = new ResizeObserver(() => setHeight(key, measureRect(element).height))
    held.set(key, { element, observer })
    observer.observe(element)
    setHeight(key, measureRect(element).height)
  }

  onScopeDispose(() => {
    for (const { observer } of held.values()) observer.disconnect()
    held.clear()
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

  return { setWidgets, getNodeRoom }
}
