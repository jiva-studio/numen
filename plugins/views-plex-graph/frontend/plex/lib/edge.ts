import type { Position } from './node'

/** The end of a line an arrowhead is drawn at, pointing out of the line there. */
export type EdgeArrow = 'from' | 'to'

export interface PlexEdge {
  readonly from: string
  readonly to: string
  /** Drawn along the line when present; the plex never invents one. */
  readonly label?: string
  /** The end an arrowhead is drawn at. A line given none carries none. */
  readonly arrow?: EdgeArrow
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
   * Where those words sit: the fraction of the line they are read along that
   * the middle of them relies on. The middle of the line, unless the title
   * had to slide along it to find room.
   */
  readonly wordsAt: number
  /** Where the arrowhead goes and which way it is aimed, for an edge with one. */
  readonly arrowhead?: PlacedArrow | undefined
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
 * How far along the curve each of two dozen points on it lies, the whole
 * length last. Every step is a chord, which is shorter than the arc it spans,
 * so the answer runs a little under.
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
 * Which `t` stands a fraction of the way along the measured length. The steps
 * are chords of even parameter, so the answer walks them and lands between the
 * two the fraction falls across.
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
 * Which way round a curve is taken for the words set on it, read off the
 * tangent at `at`, where the middle of the title sits. A curve with no
 * sideways run at all is taken down the page.
 *
 * A relation says which way an edge was routed, and the curve says something else:
 * one that loops out of a column arrives at its midpoint running back the way
 * it came, and one that doubles back turns twice over a short run.
 */
export const getHeading = (edge: EdgeCurve, pos: number): EdgeHeading => {
  const way = getTangent(edge, getParameter(measureAxis(edge), pos))
  if (way.x !== 0) return way.x < 0 ? 'against' : 'along'
  return way.y < 0 ? 'against' : 'along'
}

/** How far back from its point an arrowhead reaches, in the units it is drawn in. */
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

  return { at: point, angle: (Math.atan2(out.y, out.x) * 180) / Math.PI }
}
