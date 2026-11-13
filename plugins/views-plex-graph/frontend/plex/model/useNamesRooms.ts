/** The names set around each node, wrapped to the width of a node and measured in the type of a name. */
import { computed, watch, type ComputedRef, type Ref } from 'vue'
import { getNamesRoom, type NamesRoom } from '../lib/names'
import { NO_ROOM, type NodeRoom, type PlexOptions } from '../lib/arrange'
import type { PlexMetrics } from './sizing/measure'
import type { PlexGraph, PlexNames, PlexNode } from '../../types'

/** What the rooms of the names are made from: the graph, and the names to show with it. */
export interface NamesRoomsProps {
  readonly graph: PlexGraph
  readonly names?: ((id: string) => PlexNames) | undefined
  readonly showNames?: boolean | undefined
}

/**
 * The room of the names of each node: `getRoomOf` answers for a node with names and nothing for one without.
 * A node that has left the graph keeps the room it had until it is gone from the frame, so its names leave with its pill.
 * `room` is what the arrangement is given, and is absent while no node has names.
 */
export function useNamesRooms(
  props: NamesRoomsProps,
  measures: Readonly<Ref<PlexMetrics | undefined>>,
  options: Readonly<Ref<PlexOptions>>,
): {
  getRoomOf: (id: string) => NamesRoom | undefined
  room: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
} {
  const rooms = computed<ReadonlyMap<string, NamesRoom>>(() => {
    const found = new Map<string, NamesRoom>()
    const metrics = measures.value
    const { names } = props
    if (!props.showNames || !names || !metrics) return found
    for (const node of props.graph.nodes) {
      if (node.relation === 'focus') continue
      const room = getNamesRoom(names(node.id), {
        fontSize: metrics.nameSize,
        width: options.value.nodeSize.width,
        measure: metrics.name,
      })
      if (room.above.length > 0 || room.below.length > 0) found.set(node.id, room)
    }
    return found
  })

  let leaving: ReadonlyMap<string, NamesRoom> = new Map()
  watch(rooms, (_, previous) => {
    leaving = previous
  })
  const drawn = computed(() => new Set(props.graph.nodes.map((node) => node.id)))

  const room = computed(() => {
    const found = rooms.value
    return found.size > 0 ? (node: PlexNode) => found.get(node.id) ?? NO_ROOM : undefined
  })

  const getRoomOf = (id: string) =>
    rooms.value.get(id) ?? (drawn.value.has(id) ? undefined : leaving.get(id))

  return { getRoomOf, room }
}
