import type { PlexFrame } from '../../frame'
import type { Position } from '../../node'
import type { PlexOptions, Size } from '../options'
import { findNode, getRelationTowards } from './aim'
import type { PlexRelation } from '../../../../types'

/**
 * What dragging away from a node and dropping evaluates to.
 * The plex reports the shape of the gesture only: the node it started from, the direction, and whether it landed on something.
 * What a parent or a jump means, what gets written and whether it is allowed, is the application's.
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
 * One more child is the default; failing that, the first relation the caller allows.
 * A gesture moves nowhere when it is a press and not a drag: from a trackpad, a pointer that cannot hold a button down, or the keyboard.
 */
export const getRelationWithoutDirection = (
  relations: readonly PlexRelation[],
): PlexRelation | null => (relations.includes('child') ? 'child' : (relations[0] ?? null))

export interface DropInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  readonly from: string
  readonly point: Position
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
  const relation = getRelationTowards(source, target, options, relations)
  if (!relation) return null

  return landedOn
    ? { kind: 'link', from, to: landedOn.id, relation }
    : { kind: 'create', from, relation }
}

export interface DroppedInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  readonly viewport: Size
  readonly point: Position
  readonly relations: readonly PlexRelation[]
  readonly threshold: number
}

/**
 * The relation something dragged in from outside evaluates to.
 *
 * Measured from the focus, which is what the arrangement is built around. A
 * node the pointer crosses is not a landing: the relation is read off the
 * direction, and dropping anywhere in the window is answered the same way.
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

  return getRelationTowards(focus, point, options, relations)
}
