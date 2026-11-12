/** How much of a neighborhood is shown: the window decides, and the cap holds when there is none. */
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
 * smaller group is placed differently. Every pass removes at least one node.
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
    const placed = lay(admitted, createStopWhenSettled(admitted, isInside))
    const outside = new Set(placed.filter((node) => !isInside(node)).map((node) => node.id))
    if (outside.size === 0) return admitted
    admitted = cutBeforeOutside(admitted, outside)
  }
}

/** Stops a pass when each relation has met a box outside or has all its boxes placed. */
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
    kept[relation] = cut === -1 ? nodes : nodes.slice(0, cut)
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
