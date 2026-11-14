import type { Position, Size } from '@numen/ui'
import type { PlexNode } from '../../types'

export type { Position }

/** A node placed on the canvas, in the plex's own coordinates. */
export interface PlacedNode extends PlexNode {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
  readonly order: number
  readonly opacity: number
}

/**
 * The shape of a placeholder that is not a node yet, drawn where releasing the
 * gesture would place it. It is a box like any other, except that it has no
 * title of its own and shows the relation it would take.
 */
export const ghostNode = (node: PlexNode, point: Position, size: Size): PlacedNode => ({
  id: node.id,
  title: node.title,
  relation: node.relation,
  x: point.x,
  y: point.y,
  width: size.width,
  height: size.height,
  order: 0,
  opacity: 1,
})

/**
 * Whether a node can be chosen. One predicate, because the rule decides three
 * things: the click, the tab stop and the accessible tree.
 *
 * A node that is not fully there is on its way in or out, and choosing it
 * would pick something the reader never saw.
 */
export const isReachable = (node: PlacedNode): boolean =>
  node.relation !== 'focus' && node.opacity >= 1

/**
 * Whether the keyboard can stop on a node.
 *
 * The focused node is a stop although it cannot be chosen, because a menu is
 * requested from wherever the keyboard is.
 */
export const isStop = (node: PlacedNode): boolean => node.opacity >= 1

/**
 * What a node is to a gesture. A node has exactly one role, and one that does not exist yet has none.
 * - `open` — no gesture is in progress, so a pointer over it may start one; `closed` — a gesture cannot start here
 * - `source` — the gesture in progress started here
 * - `target` — releasing now would link the gesture to this node
 * - `ghost` — not a node yet: the shape of what releasing here would create
 */
export type GestureRole = 'open' | 'closed' | 'source' | 'target' | 'ghost'

/** A title may be empty; an accessible name may not. */
export const createAriaLabel = (node: PlexNode): string =>
  `${node.title || 'Untitled'}, ${node.relation}`

/** What a name set under a note is, and whose it is: "Rootstock list, name under Pear rows". */
export const createNameLabel = (title: string, owner: PlexNode): string =>
  `${title || 'Untitled'}, name under ${owner.title || 'Untitled'}`

/** The radius of the handle's disc, in the plex's own pixels. */
export const HANDLE_RADIUS = 9

/** The width of the outline of the handle's disc, in the plex's own pixels. */
export const HANDLE_STROKE = 1

/**
 * The handle position within a node: halfway down, on its trailing edge when the pill's outline
 * is drawn there, and with its whole disc beyond that edge when it is not, because the title
 * ends inside the padding and the disc must not cover it.
 *
 * Given the width of the window, a disc beyond the trailing edge that would leave the window is
 * set beyond the leading edge instead.
 *
 * Both drawings read it from here: the node draws the handle there, and the
 * plex starts the gesture's thread there.
 */
export const getHandle = (node: PlacedNode, hasPill = true, viewportWidth?: number): Position => {
  const edge = node.width / 2
  if (hasPill) return { x: edge, y: 0 }

  const reach = 2 * HANDLE_RADIUS + HANDLE_STROKE
  const isOutside = viewportWidth !== undefined && node.x + edge + reach > viewportWidth / 2
  return { x: isOutside ? -(edge + HANDLE_RADIUS) : edge + HANDLE_RADIUS, y: 0 }
}
