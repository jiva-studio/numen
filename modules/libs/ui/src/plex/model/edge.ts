import type { Point } from './node'

export interface PlexEdge {
  readonly from: string
  readonly to: string
  /** Drawn along the line when present; the plex never invents one. */
  readonly label?: string
}

/**
 * The line itself, as a cubic curve. Two control points, because both tangents
 * matter: an edge leaves and arrives square-on whatever sideways distance it
 * covers, and one control point cannot fix both.
 */
export interface EdgeCurve {
  readonly fromPoint: Point
  readonly toPoint: Point
  readonly control1: Point
  readonly control2: Point
}

/**
 * Which way the words of a title run along a curve. `left` is across the page
 * against reading, so the title is set on the curve taken the other way round.
 *
 * A curve running up or down the page, or turning hard under the length of a
 * word, holds no direction the words can take, and carries `none`.
 */
export type EdgeHeading = 'left' | 'right' | 'none'

/** An edge routed between two placed nodes. */
export interface PlacedEdge extends PlexEdge, EdgeCurve {
  readonly opacity: number
  readonly heading: EdgeHeading
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
export const midpointOf = (edge: EdgeCurve): Point => ({
  x: (edge.fromPoint.x + 3 * edge.control1.x + 3 * edge.control2.x + edge.toPoint.x) / 8,
  y: (edge.fromPoint.y + 3 * edge.control1.y + 3 * edge.control2.y + edge.toPoint.y) / 8,
})

/** Which way the curve is travelling at `t`, as a direction of any length. */
const tangentAt = (edge: EdgeCurve, t: number): Point => {
  const back = 1 - t
  const leaving = 3 * back * back
  const middle = 6 * back * t
  const arriving = 3 * t * t
  return {
    x:
      leaving * (edge.control1.x - edge.fromPoint.x) +
      middle * (edge.control2.x - edge.control1.x) +
      arriving * (edge.toPoint.x - edge.control2.x),
    y:
      leaving * (edge.control1.y - edge.fromPoint.y) +
      middle * (edge.control2.y - edge.control1.y) +
      arriving * (edge.toPoint.y - edge.control2.y),
  }
}

/** The angle between two directions, whichever way round it turns. */
const angleBetween = (a: Point, b: Point): number =>
  Math.abs(Math.atan2(a.x * b.y - a.y * b.x, a.x * b.x + a.y * b.y))

/** The stretch of a curve a title covers, either side of the midpoint. */
const TITLE_STRETCH = [0.25, 0.75] as const

/**
 * How far a curve may turn between its midpoint and either end of that
 * stretch. Past this the letters lean against each other and the word stops
 * being one.
 */
const TURN_LIMIT = Math.PI / 6

/**
 * Which way a curve runs where its title sits, and whether it holds that
 * direction for the length of a word.
 *
 * A seat says which way an edge was routed, and the curve says something else:
 * one that loops out of a column arrives at its midpoint running down the
 * page, and one joining two boxes closer together than the reach it leaves
 * with doubles back on itself.
 */
export const headingOf = (edge: EdgeCurve): EdgeHeading => {
  const middle = tangentAt(edge, 0.5)
  if (Math.abs(middle.y) > Math.abs(middle.x)) return 'none'

  const turn = Math.max(
    ...TITLE_STRETCH.map((t) => angleBetween(tangentAt(edge, t), middle)),
  )
  if (turn > TURN_LIMIT) return 'none'

  return middle.x < 0 ? 'left' : 'right'
}
