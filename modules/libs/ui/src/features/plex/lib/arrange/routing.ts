import {
  arrowOf,
  headingOf,
  lengthOf,
  type EdgeCurve,
  type PlacedEdge,
  type PlexEdge,
} from '../edge'
import type { PlacedNode, Position } from '../node'
import type { PlexSeat } from '../seat'
import { isVertical, type PlexOptions, type RoutingOptions } from './options'

/** `auto` means take the axis from the geometry. */
export type Axis = 'vertical' | 'horizontal' | 'auto'

export interface Routing extends RoutingOptions {
  readonly axisOf: (node: { seat: PlexSeat }) => Axis
  /**
   * How wide a title is set. Text is measured where the plex is drawn; here it
   * arrives as a number, and without a measurer the whole label is drawn.
   */
  readonly labelWidth?: ((label: string) => number) | undefined
  /**
   * How deep one line of a title stands, across the line it is set on. It is
   * measured in the type a title is set in, which is a type of its own, and is
   * nothing where nothing measured it.
   */
  readonly labelDepth: number
}

/**
 * The axis is read off the seat, not off the distance between the boxes: the
 * last child in a wide row is further sideways than it is down, and measuring
 * would send its edge out of the focus's side.
 */
export function routingFor(
  options: PlexOptions,
  measureLabel?: (label: string) => number,
  labelDepth = 0,
): Routing {
  return {
    ...options.routing,
    labelWidth: measureLabel,
    labelDepth,
    axisOf: (node) =>
      node.seat === 'focus'
        ? 'auto'
        : isVertical(options.direction[node.seat])
          ? 'vertical'
          : 'horizontal',
  }
}

/** The one character a cut title ends in. */
const ELLIPSIS = '…'

/** The middle of a line, where a title is set until something is in the way. */
export const MIDDLE = 0.5

/**
 * The words a curve has room for: the longest start of the label that fits it,
 * the ellipsis included. The prefix is found by halving, and a curve with room
 * for nothing carries the ellipsis alone.
 */
export function cutToFit(label: string, room: number, width: (label: string) => number): string {
  if (width(label) <= room) return label

  const letters = [...label]
  const truncateTo = (count: number) => `${letters.slice(0, count).join('').trimEnd()}${ELLIPSIS}`

  let fits = 0
  let over = letters.length
  while (fits + 1 < over) {
    const middle = Math.floor((fits + over) / 2)
    if (width(truncateTo(middle)) <= room) fits = middle
    else over = middle
  }
  return truncateTo(fits)
}

/** A fixed place on a border, so a row of edges reads as a fan. */
function getEdgeAnchor(node: PlacedNode, side: 'top' | 'bottom' | 'left' | 'right'): Position {
  switch (side) {
    case 'top':
      return { x: node.x, y: node.y - node.height / 2 }
    case 'bottom':
      return { x: node.x, y: node.y + node.height / 2 }
    case 'left':
      return { x: node.x - node.width / 2, y: node.y }
    case 'right':
      return { x: node.x + node.width / 2, y: node.y }
  }
}

/** Whether there is clear space between two boxes along the given axis. */
function isSeparated(a: PlacedNode, b: PlacedNode, isVertical: boolean): boolean {
  return isVertical
    ? Math.abs(b.y - a.y) > (a.height + b.height) / 2
    : Math.abs(b.x - a.x) > (a.width + b.width) / 2
}

/** Which way an edge runs: down the picture, or across it. */
function isVerticalRun(from: PlacedNode, to: PlacedNode, routing: Routing): boolean {
  const declared = [routing.axisOf(from), routing.axisOf(to)].find((axis) => axis !== 'auto')
  const wanted =
    declared === 'vertical' ||
    (declared === undefined && Math.abs(to.y - from.y) >= Math.abs(to.x - from.x))

  // Two nodes in the same row have no vertical room between their gates.
  // Joining them bottom-to-top there loops down out of one box and back up.
  return isSeparated(from, to, wanted) || !isSeparated(from, to, !wanted) ? wanted : !wanted
}

/** The control point standing out from a getEdgeAnchor along the run. */
function controlFrom(at: Position, vertical: boolean, reach: number): Position {
  return vertical ? { x: at.x, y: at.y + reach } : { x: at.x + reach, y: at.y }
}

/** The same curve read from its other end. */
function reverse(curve: EdgeCurve): EdgeCurve {
  return {
    fromPoint: curve.toPoint,
    control1: curve.control2,
    control2: curve.control1,
    toPoint: curve.fromPoint,
  }
}

/** A cubic leaving both boxes square-on, read from the caller's end. */
function curveBetween(
  from: PlacedNode,
  to: PlacedNode,
  isVertical: boolean,
  routing: Routing,
): EdgeCurve {
  const fromFirst = isVertical ? from.y <= to.y : from.x <= to.x
  const [first, second] = fromFirst ? [from, to] : [to, from]

  const firstGate = getEdgeAnchor(first, isVertical ? 'bottom' : 'right')
  const secondGate = getEdgeAnchor(second, isVertical ? 'top' : 'left')

  const span = isVertical ? secondGate.y - firstGate.y : secondGate.x - firstGate.x
  const reach = Math.max(routing.minReach, Math.abs(span) * routing.curvature)

  const along: EdgeCurve = {
    fromPoint: firstGate,
    control1: controlFrom(firstGate, isVertical, reach),
    control2: controlFrom(secondGate, isVertical, -reach),
    toPoint: secondGate,
  }

  // The caller's from and to are kept, so the curve may run right to left or
  // bottom to top.
  return fromFirst ? along : reverse(along)
}

/**
 * The words a curve carries. A title is set about the middle of its line and an
 * arrowhead sits on one end, so a line carrying one has room for fewer words.
 */
function cutLabel(edge: PlexEdge, curve: EdgeCurve, routing: Routing): string | undefined {
  if (edge.label === undefined || !routing.labelWidth) return edge.label
  const room = lengthOf(curve) - (edge.arrow ? 2 * routing.arrowRoom : 0)
  return cutToFit(edge.label, room, routing.labelWidth)
}

/**
 * Route an edge as a cubic curve leaving both boxes square-on. Dropped if
 * either endpoint is not on the canvas.
 */
export function routeEdge(
  edge: PlexEdge,
  byId: ReadonlyMap<string, PlacedNode>,
  routing: Routing,
  opacity = 1,
): PlacedEdge | null {
  const from = byId.get(edge.from)
  const to = byId.get(edge.to)
  if (!from || !to || from === to) return null

  const curve = curveBetween(from, to, isVerticalRun(from, to, routing), routing)

  return {
    ...edge,
    ...curve,
    opacity,
    heading: headingOf(curve, MIDDLE),
    words: cutLabel(edge, curve, routing),
    wordsAt: MIDDLE,
    arrowhead: edge.arrow ? arrowOf(curve, edge.arrow) : undefined,
  }
}

export function routeEdges(
  edges: readonly PlexEdge[],
  byId: ReadonlyMap<string, PlacedNode>,
  routing: Routing,
  opacity: (edge: PlexEdge) => number = () => 1,
): PlacedEdge[] {
  return edges
    .map((edge) => routeEdge(edge, byId, routing, opacity(edge)))
    .filter((edge): edge is PlacedEdge => edge !== null)
}
