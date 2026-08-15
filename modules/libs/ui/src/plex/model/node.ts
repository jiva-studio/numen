import type { PlexRole } from './role'

export interface Point {
  readonly x: number
  readonly y: number
}

/**
 * A node as the caller describes it. The id is opaque: the plex has no way to
 * ask what it addresses, and the role is a position in the picture rather than
 * a claim about a graph.
 */
export interface PlexNode {
  readonly id: string
  readonly label: string
  readonly role: PlexRole
}

/** A node placed on the canvas, in the plex's own coordinates. */
export interface PlacedNode extends PlexNode {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
  /** Rank within the node's own role, outward from the focus. */
  readonly order: number
  /** Always 1 except partway through a movement. */
  readonly opacity: number
}

/**
 * Whether a node can be chosen. One predicate, because the rule decides three
 * things: the click, the tab stop and the accessible tree.
 *
 * A node that is not fully there is on its way in or out, and choosing it
 * would pick something the reader never saw.
 */
export const isReachable = (node: PlacedNode): boolean =>
  node.role !== 'focus' && node.opacity >= 1

/**
 * What a node is to a gesture, beyond a box with a label.
 *
 * One value rather than a flag each, because a node is only ever one of these:
 * a hand cannot be reaching out of a node and aiming at it at once, and a node
 * that is not there yet is none of them.
 *
 * - `open` — nothing is under way, and a hand over it may reach out from it
 * - `closed` — reaching out from here is not on offer
 * - `source` — the gesture under way left from here
 * - `target` — letting go now would link the gesture to this node
 * - `ghost` — not a node yet: the shape of what letting go here would make
 */
export type NodeStanding = 'open' | 'closed' | 'source' | 'target' | 'ghost'

/** A label may be empty; an accessible name may not. */
export const nameOf = (node: PlexNode): string =>
  `${node.label || 'Untitled'}, ${node.role}`

/**
 * Where the handle sits within a node: on its trailing edge, halfway down.
 *
 * Here rather than in either drawing because both need the same answer — the
 * node draws the handle there, and the plex starts the gesture's thread there.
 */
export const handleIn = (node: PlacedNode): Point => ({ x: node.width / 2, y: 0 })
