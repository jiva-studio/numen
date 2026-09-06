/**
 * The curves themselves, as the `d` and `transform` strings a renderer draws.
 *
 * Where a line runs is settled in `routing.ts`; this is only how that is
 * written down.
 */
import { ARROW_LENGTH, type EdgeCurve, type PlacedArrow, type PlacedEdge } from '../edge'
import type { Position } from '../../lib/geometry'

/** The curve, from where it leaves to where it arrives. */
export const pathOf = (edge: EdgeCurve): string =>
  `M ${edge.fromPoint.x} ${edge.fromPoint.y}` +
  ` C ${edge.control1.x} ${edge.control1.y}` +
  ` ${edge.control2.x} ${edge.control2.y}` +
  ` ${edge.toPoint.x} ${edge.toPoint.y}`

/** The same curve, running the way its words are read. */
export const readingPathOf = (edge: PlacedEdge): string =>
  edge.heading === 'against'
    ? `M ${edge.toPoint.x} ${edge.toPoint.y}` +
      ` C ${edge.control2.x} ${edge.control2.y}` +
      ` ${edge.control1.x} ${edge.control1.y}` +
      ` ${edge.fromPoint.x} ${edge.fromPoint.y}`
    : pathOf(edge)

/** The head itself, drawn about its own tip and pointing along the x axis. */
export const ARROWHEAD_PATH = `M 0 0 L ${-ARROW_LENGTH} 5.5 L ${-ARROW_LENGTH} -5.5 Z`

/** The head put on its end of the line and turned along it. */
export const arrowTransformOf = (arrow: PlacedArrow): string =>
  `translate(${arrow.at.x} ${arrow.at.y}) rotate(${arrow.angle})`

/**
 * A line between two loose points: out of the first and into the second, each
 * end square-on, reaching half the sideways distance between them.
 */
export const threadOf = (start: Position, to: Position): string => {
  const reachOut = Math.abs(to.x - start.x) / 2
  return (
    `M ${start.x} ${start.y}` +
    ` C ${start.x + reachOut} ${start.y} ${to.x - reachOut} ${to.y} ${to.x} ${to.y}`
  )
}
