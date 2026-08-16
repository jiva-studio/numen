import type { Point } from './node'

export interface PlexEdge {
  readonly from: string
  readonly to: string
  /** Drawn along the line when present; the plex never invents one. */
  readonly label?: string
}

/**
 * An edge routed between two placed nodes, as a cubic curve. Two control
 * points, because both tangents matter: an edge leaves and arrives square-on
 * whatever sideways distance it covers, and one control point cannot fix both.
 */
export interface PlacedEdge extends PlexEdge {
  readonly fromPoint: Point
  readonly toPoint: Point
  readonly control1: Point
  readonly control2: Point
  readonly opacity: number
}

/**
 * An edge is the pair it joins; two lines between the same pair are one, and
 * which end is which is not part of that.
 *
 * A relationship that is the same in both directions changes ends when the plex
 * moves along it — what was a jump into the focus is a jump out of it from the
 * other side. Reading those as two edges fades one out while the other fades
 * in, so the one line the reader is following is the one that disappears.
 */
export const edgeKey = (edge: PlexEdge): string =>
  edge.from < edge.to ? `${edge.from} ${edge.to}` : `${edge.to} ${edge.from}`

/** Halfway along the curve, where a label sits clear of the line. */
export const midpointOf = (edge: PlacedEdge): Point => ({
  x: (edge.fromPoint.x + 3 * edge.control1.x + 3 * edge.control2.x + edge.toPoint.x) / 8,
  y: (edge.fromPoint.y + 3 * edge.control1.y + 3 * edge.control2.y + edge.toPoint.y) / 8,
})
