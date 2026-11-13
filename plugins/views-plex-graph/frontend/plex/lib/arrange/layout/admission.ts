/** How much of a graph is shown: the window decides, and the cap holds when there is none. */
import { RELATION_NAMES } from '../../relation'
import type { PlacedNode } from '../../node'
import type { PlexOptions } from '../options'
import type { NodesByRelation } from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** The nodes of each relation, in the order they arrived. Without a window each relation is cut to `maxPerCluster`. */
export function requestNodes(nodes: readonly PlexNode[], options: PlexOptions): NodesByRelation {
  const requested: Partial<Record<PlexRelation, readonly PlexNode[]>> = {}
  for (const relation of RELATION_NAMES) {
    const ofRelation = nodes.filter((node) => node.relation === relation)
    if (ofRelation.length === 0) continue
    requested[relation] = options.viewport ? ofRelation : ofRelation.slice(0, options.maxPerCluster)
  }
  return requested
}

/**
 * The nodes the window holds. The set is laid out, each relation is cut before
 * the first box that is not inside, and the rest is laid out again, since a
 * smaller group is placed differently. A group whose first box is outside is
 * halved instead, since its middle lies nearer the focus when it is smaller.
 * Every pass removes at least one node. What was cut is then taken back, one note of each relation in turn, while the window still holds every box.
 *
 * A pass needs the first box outside of each relation and nothing after it, so
 * it stops once every relation has one or is fully placed.
 */
export function admitNodes(
  requested: NodesByRelation,
  lay: (
    nodesByRelation: NodesByRelation,
    shouldStop: (node: PlacedNode) => boolean,
  ) => readonly PlacedNode[],
  isInside: (node: PlacedNode) => boolean,
): NodesByRelation {
  let admitted = requested
  for (;;) {
    const outside = findOutside(admitted, lay, isInside)
    if (outside.size === 0) break
    admitted = cutBeforeOutside(admitted, outside)
  }
  return refill(
    admitted,
    requested,
    (nodesByRelation) => findOutside(nodesByRelation, lay, isInside).size === 0,
  )
}

function findOutside(
  nodesByRelation: NodesByRelation,
  lay: (
    nodesByRelation: NodesByRelation,
    shouldStop: (node: PlacedNode) => boolean,
  ) => readonly PlacedNode[],
  isInside: (node: PlacedNode) => boolean,
): ReadonlySet<string> {
  const placed = lay(nodesByRelation, createStopWhenSettled(nodesByRelation, isInside))
  return new Set(placed.filter((node) => !isInside(node)).map((node) => node.id))
}

function refill(
  admitted: NodesByRelation,
  requested: NodesByRelation,
  isHeld: (nodesByRelation: NodesByRelation) => boolean,
): NodesByRelation {
  let current = admitted
  for (let grew = true; grew;) {
    grew = false
    for (const relation of RELATION_NAMES) {
      const all = requested[relation] ?? []
      const have = current[relation]?.length ?? 0
      if (have >= all.length) continue
      const next = { ...current, [relation]: all.slice(0, have + 1) }
      if (isHeld(next)) {
        current = next
        grew = true
      }
    }
  }
  return current
}

function createStopWhenSettled(
  nodesByRelation: NodesByRelation,
  isInside: (node: PlacedNode) => boolean,
): (node: PlacedNode) => boolean {
  const unsettled = new Map<string, number>()
  for (const relation of RELATION_NAMES) {
    const count = nodesByRelation[relation]?.length ?? 0
    if (count > 0) unsettled.set(relation, count)
  }
  const isSettled = new Set<string>()
  return (node) => {
    const relation = node.relation
    if (isSettled.has(relation)) return false
    const left = (unsettled.get(relation) ?? 1) - 1
    unsettled.set(relation, left)
    if (left === 0 || !isInside(node)) {
      isSettled.add(relation)
      unsettled.delete(relation)
    }
    return unsettled.size === 0
  }
}

function cutBeforeOutside(nodesByRelation: NodesByRelation, outside: ReadonlySet<string>) {
  const kept: Partial<Record<PlexRelation, readonly PlexNode[]>> = {}
  for (const relation of RELATION_NAMES) {
    const nodes = nodesByRelation[relation]
    if (!nodes) continue
    const cut = nodes.findIndex((node) => outside.has(node.id))
    // The middle of a group lies further out the more it holds, so when even the
    // first box is outside a smaller group may still fit: halve it and lay again.
    kept[relation] = cut === -1 ? nodes : nodes.slice(0, cut === 0 ? nodes.length >> 1 : cut)
  }
  return kept
}

/** How many of each relation arrived and were not admitted, for the relations where that is above nothing. */
export function countOverflow(
  nodes: readonly PlexNode[],
  admitted: NodesByRelation,
): Partial<Record<PlexRelation, number>> {
  const overflow: Partial<Record<PlexRelation, number>> = {}
  for (const relation of RELATION_NAMES) {
    const arrived = nodes.filter((node) => node.relation === relation).length
    const left = arrived - (admitted[relation]?.length ?? 0)
    if (left > 0) overflow[relation] = left
  }
  return overflow
}
