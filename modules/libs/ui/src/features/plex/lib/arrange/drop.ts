import type { PlexFrame } from '../frame'
import type { PlacedNode, Position } from '../node'
import { RELATION_NAMES, type PlexRelation } from '../relation'
import type { Direction, PlexOptions, Size } from './options'

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
 * The relation a gesture that went nowhere asks for.
 *
 * One more child, because that is what anyone reaches for; failing that, the
 * first relation the caller allows. A gesture goes nowhere when it is a press
 * and not a drag — from a trackpad, from a hand that cannot hold a button
 * down, or from the keyboard, where there is no direction to read at all.
 */
export const getRelationWithoutDirection = (
  relations: readonly PlexRelation[],
): PlexRelation | null => (relations.includes('child') ? 'child' : (relations[0] ?? null))

/**
 * Which way a point lies from another.
 *
 * Not simply the axis it lies getFurthest along: rows are wide and columns are
 * bias of one, the outermost child of a wide row is further sideways than it
 * is down, and dragging towards where the children plainly are would name
 * something off to the side.
 */
function getDropDirection(dx: number, dy: number, bias: number): Direction | null {
  if (dx === 0 && dy === 0) return null
  if (Math.abs(dy) * bias >= Math.abs(dx)) {
    return dy < 0 ? 'up' : 'down'
  }
  return dx < 0 ? 'left' : 'right'
}

/**
 * The relation a direction stands for — the inverse of the arrangement's own map.
 *
 * It is read off `direction`, which is what put the relations where they are: with
 * parents sent down, dragging down means a parent.
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
export function findNode(at: Position, frame: PlexFrame): PlacedNode | null {
  return (
    [...frame.nodes]
      .reverse()
      .find(
        (node) =>
          node.opacity >= 1 &&
          Math.abs(at.x - node.x) <= node.width / 2 &&
          Math.abs(at.y - node.y) <= node.height / 2,
      ) ?? null
  )
}

export interface DropInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  /** The node the gesture started from. */
  readonly from: string
  /** Where it was let go, in the plex's own coordinates. */
  readonly at: Position
  /** Relations a gesture is allowed to produce. */
  readonly relations: readonly PlexRelation[]
}

/**
 * Landing on another node makes a link; landing on nothing makes a node. That
 * one rule is also what keeps a gesture from a node other than the focus
 * workable: the relations around a child overlap the rest of the picture, and
 * whatever the gesture crosses, only where it stops decides.
 */
export function resolveDrop({ frame, options, from, at, relations }: DropInput): Drop | null {
  const source = frame.nodes.find((node) => node.id === from)
  if (!source) return null

  const landedOn = findNode(at, frame)
  const target = landedOn ? { x: landedOn.x, y: landedOn.y } : at
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
  readonly at: Position
  /** Relations a gesture is allowed to produce. */
  readonly relations: readonly PlexRelation[]
  /** How far from the focus the pointer stands before it names a direction. */
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
  at,
  relations,
  threshold,
}: DroppedInput): PlexRelation | null {
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!focus) return null

  // Past the edge of the window is over whatever is drawn beside the plex.
  if (Math.abs(at.x) > viewport.width / 2 || Math.abs(at.y) > viewport.height / 2) return null

  // Close enough to the focus that the direction is the tremor of a hand.
  if (Math.hypot(at.x - focus.x, at.y - focus.y) < threshold) return null

  const relation = getRelationTowards(focus, at, options)
  return relation && relations.includes(relation) ? relation : null
}
