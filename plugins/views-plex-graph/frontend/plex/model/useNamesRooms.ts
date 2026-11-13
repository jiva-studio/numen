/** The names set around each node, wrapped to the width of a node and measured in the type of a name. */
import { computed, type ComputedRef } from 'vue'
import { getNamesRoom, type NamesRoom } from '../lib/names'
import { NO_ROOM, type NodeRoom } from '../lib/arrange'
import type { PlexMetrics } from './sizing/measure'
import type { PlexGraph, PlexNames, PlexNode } from '../../types'

/** What the rooms of the names are made from. */
export interface NamesRoomsInput {
  readonly graph: () => PlexGraph
  readonly names: () => ((id: string) => PlexNames) | undefined
  readonly isShown: () => boolean
  readonly metrics: () => PlexMetrics | undefined
  readonly width: () => number
}

/**
 * The room of the names of each node, by node. A node with none has no entry.
 * `room` is what the arrangement is given, and is absent while no node has names.
 */
export function useNamesRooms(input: NamesRoomsInput): {
  rooms: ComputedRef<ReadonlyMap<string, NamesRoom>>
  room: ComputedRef<((node: PlexNode) => NodeRoom) | undefined>
} {
  const rooms = computed<ReadonlyMap<string, NamesRoom>>(() => {
    const found = new Map<string, NamesRoom>()
    const metrics = input.metrics()
    const getNames = input.names()
    if (!input.isShown() || !getNames || !metrics) return found
    for (const node of input.graph().nodes) {
      if (node.relation === 'focus') continue
      const room = getNamesRoom(getNames(node.id), {
        fontSize: metrics.nameSize,
        width: input.width(),
        measure: metrics.name,
      })
      if (room.above.length > 0 || room.below.length > 0) found.set(node.id, room)
    }
    return found
  })

  const room = computed(() => {
    const found = rooms.value
    return found.size > 0 ? (node: PlexNode) => found.get(node.id) ?? NO_ROOM : undefined
  })

  return { rooms, room }
}
