import type { PlexFrame } from '../../frame'
import type { PlacedNode, Position } from '../../node'
import { RELATION_NAMES } from '../../relation'
import type { Direction, PlexOptions } from '../options'
import type { PlexRelation } from '../../../../types'

function getDropDirection(dx: number, dy: number, bias: number): Direction | null {
  if (dx === 0 && dy === 0) return null
  if (Math.abs(dy) * bias >= Math.abs(dx)) {
    return dy < 0 ? 'up' : 'down'
  }
  return dx < 0 ? 'left' : 'right'
}

/**
 * The relation a direction maps to: the inverse of the arrangement's own map.
 *
 * It is derived from `direction`, which places the relations: with parents
 * placed downward, dragging down means a parent. Where several relations share
 * a heading, the first of `relations` wins, so the answer does not depend on
 * the order of the relation table.
 */
export function getRelationTowards(
  from: Position,
  to: Position,
  options: PlexOptions,
  relations: readonly PlexRelation[] = RELATION_NAMES,
): PlexRelation | null {
  const heading = getDropDirection(to.x - from.x, to.y - from.y, options.gesture.verticalBias)
  return relations.find((relation) => options.direction[relation] === heading) ?? null
}

/**
 * The node a point falls inside, if any. Later nodes win, as when drawn.
 *
 * A node that is not fully there is not a node to drop on: it is on its way
 * in or out, and it would be a link to something the reader never saw. That is
 * the same rule that decides the click and the tab stop.
 */
export function findNode(point: Position, frame: PlexFrame): PlacedNode | null {
  return (
    [...frame.nodes]
      .reverse()
      .find(
        (node) =>
          node.opacity >= 1 &&
          Math.abs(point.x - node.x) <= node.width / 2 &&
          Math.abs(point.y - node.y) <= node.height / 2,
      ) ?? null
  )
}
