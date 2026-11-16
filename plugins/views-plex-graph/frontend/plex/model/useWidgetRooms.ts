/** The room each node keeps over and under its pill for the widgets of plugins, as their measured height. */
import { computed, type ComputedRef, type Ref } from 'vue'
import type { NodeRoom, PlexOptions } from '../lib/arrange'
import type { PlexNode } from '../../types'

export type WidgetSide = 'above' | 'below'

export function useWidgetRooms(options: Readonly<Ref<PlexOptions>>): {
  setWidgets: (id: string, side: WidgetSide, element: Element | null) => void
  getNodeRoom: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
} {
  void options
  return { setWidgets: () => undefined, getNodeRoom: computed(() => undefined) }
}
