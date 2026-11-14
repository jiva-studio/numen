import type { NamesRoom } from '../../lib/names'
import type { PlacedNode } from '../../lib/node'

/** A node with names, and the room they take. */
export interface NamedNode {
  readonly node: PlacedNode
  readonly room: NamesRoom
}

/** The nodes that have a room for names, in the order they are drawn. */
export function listNamedNodes(
  nodes: readonly PlacedNode[],
  roomOf: ((id: string) => NamesRoom | undefined) | undefined,
): NamedNode[] {
  return nodes.flatMap((node) => {
    const room = roomOf?.(node.id)
    return room ? [{ node, room }] : []
  })
}
