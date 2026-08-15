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

/** An edge is the pair it joins; two lines between the same pair are one. */
export const edgeKey = (edge: PlexEdge): string => `${edge.from} ${edge.to}`

/** Halfway along the curve, where a label sits clear of the line. */
export const midpointOf = (edge: PlacedEdge): Point => ({
  x: (edge.fromPoint.x + 3 * edge.control1.x + 3 * edge.control2.x + edge.toPoint.x) / 8,
  y: (edge.fromPoint.y + 3 * edge.control1.y + 3 * edge.control2.y + edge.toPoint.y) / 8,
})
