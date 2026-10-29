import type { PlexFrame } from '../../frame'
import type { PlacedNode, Position } from '../../node'
import { RELATION_NAMES, type PlexRelation } from '../../relation'
import type { Direction, PlexOptions, Size } from '../options'

/**
 * What dragging away from a node and letting go evaluates to.
 *
 * The plex works out the shape of the gesture and nothing else: which node it
 * started from, which way it went, and whether it landed on something. What a
 * parent or a jump then *means* — what gets written, and whether it is allowed
 * — is the application's. This reports.
 */
export type Drop =
  | { readonly kind: 'create'; readonly from: string; readonly relation: PlexRelation }
  | {
      readonly kind: 'link'
      readonly from: string
      readonly to: string
      readonly relation: PlexRelation
    }

/**
 * The relation requested by a gesture that moved nowhere.
 *
 * One more child is the default; failing that, the first relation the caller
 * allows. A gesture moves nowhere when it is a press and not a drag: from a
 * trackpad, from a pointer that cannot hold a button down, or from the
 * keyboard, where there is no direction to read at all.
 */
export const getRelationWithoutDirection = (
  relations: readonly PlexRelation[],
): PlexRelation | null => (relations.includes('child') ? 'child' : (relations[0] ?? null))

/**
 * The direction from one point to another.
 *
 * The axis with the larger distance does not decide it: rows are wide and
 * columns are narrow, so the outermost child of a wide row is further sideways
 * than it is down, and a drag towards the children would give a sideways
 * direction. `bias` weights the vertical distance.
 */
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
 * placed downward, dragging down means a parent.
 */
export function getRelationTowards(
  from: Position,
  to: Position,
  options: PlexOptions,
): PlexRelation | null {
  const heading = getDropDirection(to.x - from.x, to.y - from.y, options.gesture.verticalBias)
  return RELATION_NAMES.find((relation) => options.direction[relation] === heading) ?? null
}

/**
 * The node a point falls inside, if any. Later nodes win, as when drawn.
 *
 * A node that is not fully there is not a node to let go on: it is on its way
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

export interface DropInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  /** The node the gesture started from. */
  readonly from: string
  /** Where it was let go, in the plex's own coordinates. */
  readonly point: Position
  /** Relations a gesture is allowed to produce. */
  readonly relations: readonly PlexRelation[]
}

/**
 * Landing on another node creates a link; landing on nothing creates a node.
 * That one rule also keeps a gesture from a node other than the focus usable:
 * the relations around a child overlap the rest of the graph, and whatever the
 * gesture crosses, only where it ends decides.
 */
export function resolveDrop({ frame, options, from, point, relations }: DropInput): Drop | null {
  const source = frame.nodes.find((node) => node.id === from)
  if (!source) return null

  const landedOn = findNode(point, frame)
  const target = landedOn ? { x: landedOn.x, y: landedOn.y } : point
  const relation = getRelationTowards(source, target, options)
  if (!relation || !relations.includes(relation)) return null

  return landedOn
    ? { kind: 'link', from, to: landedOn.id, relation }
    : { kind: 'create', from, relation }
}

export interface DroppedInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  /** The window the plex is drawn in, centred on the focus. */
  readonly viewport: Size
  /** Where the pointer is, in the plex's own coordinates. */
  readonly point: Position
  /** Relations a gesture is allowed to produce. */
  readonly relations: readonly PlexRelation[]
  /** Distance from the focus the pointer must exceed to name a direction. */
  readonly threshold: number
}

/**
 * The relation something dragged in from outside evaluates to.
 *
 * Measured from the focus, which is what the arrangement is built around. A
 * node the pointer crosses is not a landing: the relation is read off the
 * direction, and letting go anywhere in the window is answered the same way.
 */
export function getDropRelation({
  frame,
  options,
  viewport,
  point,
  relations,
  threshold,
}: DroppedInput): PlexRelation | null {
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!focus) return null

  // Past the edge of the window is over whatever is drawn beside the plex.
  if (Math.abs(point.x) > viewport.width / 2 || Math.abs(point.y) > viewport.height / 2) return null

  // Close enough to the focus that the direction is pointer jitter.
  if (Math.hypot(point.x - focus.x, point.y - focus.y) < threshold) return null

  const relation = getRelationTowards(focus, point, options)
  return relation && relations.includes(relation) ? relation : null
}
