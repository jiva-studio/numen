import { RELATION_NAMES } from '../../relation'
import type { PlexOptions } from '../options'
import type { NodesByRelation, NodeRoom, Rooms } from './placement'

/**
 * Every note of a side keeps the most room over its pill that a note of that side keeps, so their pills line up.
 * The room under a pill moves no pill, so a note keeps only its own: one whose views draw nothing under it takes none.
 */
export function alignRooms(
  nodesByRelation: NodesByRelation,
  options: PlexOptions,
  room: Rooms,
): Rooms {
  const topBySide = new Map<string, number>()
  const sides = new Map<string, string>()
  for (const relation of RELATION_NAMES) {
    const side = options.direction[relation]
    for (const node of nodesByRelation[relation] ?? []) {
      topBySide.set(side, Math.max(topBySide.get(side) ?? 0, room(node).top))
      sides.set(node.id, side)
    }
  }
  return (node): NodeRoom => {
    const own = room(node)
    const top = topBySide.get(sides.get(node.id) ?? '')
    return top === undefined ? own : { ...own, top }
  }
}
