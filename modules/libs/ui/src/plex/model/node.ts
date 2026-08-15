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

/** A label may be empty; an accessible name may not. */
export const nameOf = (node: PlexNode): string =>
  `${node.label || 'Untitled'}, ${node.role}`
