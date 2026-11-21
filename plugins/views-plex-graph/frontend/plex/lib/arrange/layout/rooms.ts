import { RELATION_NAMES } from '../../relation'
import type { PlexOptions } from '../options'
import type { NodesByRelation, NodeRoom, Rooms } from './placement'

/** Every note of a side keeps the most room over its pill and the most under it that a note of that side keeps, so their pills line up. */
export function alignRooms(
  nodesByRelation: NodesByRelation,
  options: PlexOptions,
  room: Rooms,
): Rooms {
  const bySide = new Map<string, { top: number; bottom: number }>()
  const sides = new Map<string, string>()
  for (const relation of RELATION_NAMES) {
    const side = options.direction[relation]
    for (const node of nodesByRelation[relation] ?? []) {
      const { top, bottom } = room(node)
      const found = bySide.get(side) ?? { top: 0, bottom: 0 }
      bySide.set(side, { top: Math.max(found.top, top), bottom: Math.max(found.bottom, bottom) })
      sides.set(node.id, side)
    }
  }
  return (node): NodeRoom => {
    const own = room(node)
    const shared = bySide.get(sides.get(node.id) ?? '')
    return shared ? { ...shared, width: own.width } : own
  }
}
