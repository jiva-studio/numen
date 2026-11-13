/**
 * The curves themselves, as the `d` and `transform` strings a renderer draws.
 *
 * Where a line runs is settled in `routing.ts`; this is only how that is
 * written down.
 */
import { ARROW_LENGTH, type EdgeCurve, type EdgeHeading, type PlacedArrow } from '../../edge'
import type { Position } from '@numen/ui'
import { DEFAULT_OPTIONS } from '../options'

/** The curve, from where it leaves to where it arrives. */
export const createPath = (edge: EdgeCurve): string =>
  `M ${edge.fromPoint.x} ${edge.fromPoint.y}` +
  ` C ${edge.control1.x} ${edge.control1.y}` +
  ` ${edge.control2.x} ${edge.control2.y}` +
  ` ${edge.toPoint.x} ${edge.toPoint.y}`

/** The same curve, running the way the words of a title are read. */
export const createReadingPath = (edge: EdgeCurve, heading: EdgeHeading): string =>
  heading === 'against'
    ? `M ${edge.toPoint.x} ${edge.toPoint.y}` +
      ` C ${edge.control2.x} ${edge.control2.y}` +
      ` ${edge.control1.x} ${edge.control1.y}` +
      ` ${edge.fromPoint.x} ${edge.fromPoint.y}`
    : createPath(edge)

/** The head itself, drawn about its own tip and pointing along the x axis. */
export const ARROWHEAD_PATH = `M 0 0 L ${-ARROW_LENGTH} 5.5 L ${-ARROW_LENGTH} -5.5 Z`

/** The head put on its end of the line and turned along it. */
export const createArrowTransform = (arrow: PlacedArrow): string =>
  `translate(${arrow.at.x} ${arrow.at.y}) rotate(${arrow.angle})`

/**
 * A line between two loose points: out of the first and into the second, each
 * end square-on. At the default curvature it reaches half the sideways
 * distance between them; the reach scales with the curvature like the bow of
 * a line, and is nothing at 0, which makes the line straight.
 */
export const createThreadPath = (
  start: Position,
  to: Position,
  curvature: number = DEFAULT_OPTIONS.routing.curvature,
): string => {
  const reachOut = (Math.abs(to.x - start.x) / 2) * (curvature / DEFAULT_OPTIONS.routing.curvature)
  return (
    `M ${start.x} ${start.y}` +
    ` C ${start.x + reachOut} ${start.y} ${to.x - reachOut} ${to.y} ${to.x} ${to.y}`
  )
}
