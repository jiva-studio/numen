/** The room each node keeps around its pill, for its names and for the widgets of plugins together. */
import { computed, type ComputedRef, type Ref } from 'vue'
import { NO_ROOM, type NodeRoom, type PlexOptions } from '../lib/arrange'
import type { NamesRoom } from '../lib/names'
import type { PlexMetrics } from './sizing/measure'
import { useNamesRooms, type NamesRoomsProps } from './useNamesRooms'
import { useWidgetRooms, type WidgetSide } from './useWidgetRooms'
import type { PlexNode } from '../../types'

export function useRooms(
  props: NamesRoomsProps,
  measures: Readonly<Ref<PlexMetrics | undefined>>,
  options: Readonly<Ref<PlexOptions>>,
): {
  getRoomOf: (id: string) => NamesRoom | undefined
  setWidgets: (id: string, side: WidgetSide, element: Element | null) => void
  getNodeRoom: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
  setReady: (id: string, side: WidgetSide, isReady: boolean | null) => void
  isSettled: (ids: readonly string[]) => boolean
} {
  const { getRoomOf, getNodeRoom: getNamesRoom } = useNamesRooms(props, measures, options)
  const { setWidgets, getNodeRoom: getWidgetsRoom, setReady, isSettled } = useWidgetRooms(options)

  const getNodeRoom = computed(() => {
    const ofNames = getNamesRoom.value
    const ofWidgets = getWidgetsRoom.value
    if (!ofNames || !ofWidgets) return ofNames ?? ofWidgets
    return (node: PlexNode): NodeRoom => {
      const names = ofNames(node)
      const widgets = ofWidgets(node)
      if (widgets === NO_ROOM) return names
      return {
        top: names.top + widgets.top,
        bottom: names.bottom + widgets.bottom,
        width: Math.max(names.width, widgets.width),
      }
    }
  })

  return { getRoomOf, setWidgets, getNodeRoom, setReady, isSettled }
}
