import type { Position } from './node'

/** The end of a line an arrowhead is drawn at, pointing out of the line there. */
export type EdgeArrow = 'from' | 'to'

export interface PlexEdge {
  readonly from: string
  readonly to: string
  /** Drawn along the line when present; the plex never invents one. */
  readonly label?: string
  /** The ends an arrowhead is drawn at. A line given none carries none. */
  readonly arrows?: readonly EdgeArrow[]
}

/**
 * The line itself, as a cubic curve. Two control points, because both tangents
 * matter: an edge leaves and arrives square-on whatever sideways distance it
 * covers, and one control point cannot fix both.
 */
export interface EdgeCurve {
  readonly fromPoint: Position
  readonly toPoint: Position
  readonly control1: Position
  readonly control2: Position
}

/**
 * Which way round a curve the words of a title are set. `against` is a curve
 * running right to left or up the page, and its title is set on the curve
 * taken the other way round, so the words are never upside down.
 */
export type EdgeHeading = 'along' | 'against'

/**
 * Rendered arrowhead geometry: position on the curve and rotation angle
 * in degrees clockwise from the positive x axis.
 */
export interface PlacedArrow {
  /** The end of the line the arrowhead is drawn at. */
  readonly end: EdgeArrow
  readonly at: Position
  readonly angle: number
}

/** An edge routed between two placed nodes. */
export interface PlacedEdge extends PlexEdge, EdgeCurve {
  readonly opacity: number
  readonly heading: EdgeHeading
  /**
   * The label as it is drawn: cut to the length of the curve and ended in an
   * ellipsis. The whole of it where nothing measured the words.
   */
  readonly words?: string | undefined
  /**
   * Position of the label's midpoint as a fraction of the line's length: the
   * middle of the line, unless the title was shifted along it to find space.
   */
  readonly wordsAt: number
  /** Where each arrowhead goes and which way it is aimed, for an edge with any. */
  readonly arrowheads?: readonly PlacedArrow[] | undefined
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

/** Where the curve has got to at `t`. */
const getPoint = (edge: EdgeCurve, t: number): Position => {
  const back = 1 - t
  const leaving = back * back * back
  const first = 3 * back * back * t
  const second = 3 * back * t * t
  const incoming = t * t * t
  return {
    x:
      leaving * edge.fromPoint.x +
      first * edge.control1.x +
      second * edge.control2.x +
      incoming * edge.toPoint.x,
    y:
      leaving * edge.fromPoint.y +
      first * edge.control1.y +
      second * edge.control2.y +
      incoming * edge.toPoint.y,
  }
}

/** How finely the curve is cut up to be measured. */
const LENGTH_SAMPLES = 24

/**
 * Cumulative curve length at each of 24 sample points, the full length last.
 * Each step is a chord, shorter than the arc it spans, so the result slightly
 * underestimates.
 */
function measureAxis(edge: EdgeCurve): number[] {
  const axisOffset = [0]
  let previous = edge.fromPoint
  for (let step = 1; step <= LENGTH_SAMPLES; step += 1) {
    const point = getPoint(edge, step / LENGTH_SAMPLES)
    axisOffset.push(axisOffset[step - 1]! + Math.hypot(point.x - previous.x, point.y - previous.y))
    previous = point
  }
  return axisOffset
}

/** How long the curve is, as an estimate. */
export const getLength = (edge: EdgeCurve): number => measureAxis(edge)[LENGTH_SAMPLES]!

/**
 * The parameter `t` at the given fraction of the measured length. The steps are
 * chords of even parameter spacing, so the function walks them and interpolates
 * within the step the fraction falls in.
 */
function getParameter(offsets: readonly number[], fraction: number): number {
  const total = offsets[LENGTH_SAMPLES]!
  if (total <= 0) return 0
  const wanted = Math.min(Math.max(fraction, 0), 1) * total

  let step = 1
  while (step < LENGTH_SAMPLES && offsets[step]! < wanted) step += 1

  const start = offsets[step - 1]!
  const span = offsets[step]! - start
  const within = span <= 0 ? 0 : (wanted - start) / span
  return (step - 1 + within) / LENGTH_SAMPLES
}

/**
 * A ruler along the curve: where it has got to a fraction of the way along its
 * length. The curve is measured once and read many times, since a title is
 * tried at several places on the same line.
 */
export function createPointOnCurve(edge: EdgeCurve): (fraction: number) => Position {
  const axisEdge = measureAxis(edge)
  return (fraction) => getPoint(edge, getParameter(axisEdge, fraction))
}

/** Which way the curve is travelling at `t`, as a direction of any length. */
const getTangent = (edge: EdgeCurve, t: number): Position => {
  const back = 1 - t
  const leaving = 3 * back * back
  const middle = 6 * back * t
  const incoming = 3 * t * t
  return {
    x:
      leaving * (edge.control1.x - edge.fromPoint.x) +
      middle * (edge.control2.x - edge.control1.x) +
      incoming * (edge.toPoint.x - edge.control2.x),
    y:
      leaving * (edge.control1.y - edge.fromPoint.y) +
      middle * (edge.control2.y - edge.control1.y) +
      incoming * (edge.toPoint.y - edge.control2.y),
  }
}

/**
 * Direction in which a curve is traversed for the label set on it, taken from
 * the tangent at `at`, the midpoint of the title. A curve with no horizontal run
 * is traversed downward.
 *
 * The relation gives the edge's routing direction, which can differ from the
 * curve's: one that loops out of a column arrives at its midpoint running back
 * the way it came, and one that doubles back turns twice over a short run.
 */
export const getHeading = (edge: EdgeCurve, pos: number): EdgeHeading => {
  const way = getTangent(edge, getParameter(measureAxis(edge), pos))
  if (way.x !== 0) return way.x < 0 ? 'against' : 'along'
  return way.y < 0 ? 'against' : 'along'
}

/** How far back from its point an arrowhead extends, in drawing units. */
export const ARROW_LENGTH = 15

/**
 * Computes rendered position and rotation angle for an arrowhead at the specified edge endpoint.
 */
export const getArrow = (edge: EdgeCurve, end: EdgeArrow): PlacedArrow => {
  const t = end === 'to' ? 1 : 0
  const point = getPoint(edge, t)

  const axisEdge = measureAxis(edge)
  const total = axisEdge[LENGTH_SAMPLES]!
  const covered = total > ARROW_LENGTH ? ARROW_LENGTH / total : 0
  const precedingPoint = getPoint(
    edge,
    getParameter(axisEdge, end === 'to' ? 1 - covered : covered),
  )

  const spanned = { x: point.x - precedingPoint.x, y: point.y - precedingPoint.y }
  const way = getTangent(edge, t)
  const tangent = end === 'to' ? way : { x: -way.x, y: -way.y }
  const out = covered > 0 && (spanned.x !== 0 || spanned.y !== 0) ? spanned : tangent

  return { end, at: point, angle: (Math.atan2(out.y, out.x) * 180) / Math.PI }
}
