import type { PlacedEdge, PlacedNode, PlexEdge, PlexSeat, Point } from '../model'
import { isVertical, type PlexOptions, type RoutingOptions } from './options'

/** `auto` means take the axis from the geometry. */
export type Axis = 'vertical' | 'horizontal' | 'auto'

export interface Routing extends RoutingOptions {
  readonly axisOf: (node: { seat: PlexSeat }) => Axis
}

/**
 * The axis is read off the seat, not off the distance between the boxes: the
 * last child in a wide row is further sideways than it is down, and measuring
 * would send its edge out of the focus's side.
 */
export function routingFor(options: PlexOptions): Routing {
  return {
    ...options.routing,
    axisOf: (node) =>
      node.seat === 'focus'
        ? 'auto'
        : isVertical(options.direction[node.seat])
          ? 'vertical'
          : 'horizontal',
  }
}

/** A fixed point on a border, so a row of edges reads as a fan. */
function gate(node: PlacedNode, side: 'top' | 'bottom' | 'left' | 'right'): Point {
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
function separated(a: PlacedNode, b: PlacedNode, vertical: boolean): boolean {
  return vertical
    ? Math.abs(b.y - a.y) > (a.height + b.height) / 2
    : Math.abs(b.x - a.x) > (a.width + b.width) / 2
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

  const declared = [routing.axisOf(from), routing.axisOf(to)].find(
    (axis) => axis !== 'auto',
  )
  const wanted =
    declared === 'vertical' ||
    (declared === undefined && Math.abs(to.y - from.y) >= Math.abs(to.x - from.x))

  // Two nodes in the same row have no vertical room between their gates.
  // Joining them bottom-to-top there loops down out of one box and back up.
  const vertical =
    separated(from, to, wanted) || !separated(from, to, !wanted) ? wanted : !wanted

  const fromFirst = vertical ? from.y <= to.y : from.x <= to.x
  const [first, second] = fromFirst ? [from, to] : [to, from]

  const firstGate = gate(first, vertical ? 'bottom' : 'right')
  const secondGate = gate(second, vertical ? 'top' : 'left')

  const span = vertical ? secondGate.y - firstGate.y : secondGate.x - firstGate.x
  const reach = Math.max(routing.minReach, Math.abs(span) * routing.curvature)

  const firstControl: Point = vertical
    ? { x: firstGate.x, y: firstGate.y + reach }
    : { x: firstGate.x + reach, y: firstGate.y }
  const secondControl: Point = vertical
    ? { x: secondGate.x, y: secondGate.y - reach }
    : { x: secondGate.x - reach, y: secondGate.y }

  return fromFirst
    ? {
        ...edge,
        fromPoint: firstGate,
        control1: firstControl,
        control2: secondControl,
        toPoint: secondGate,
        opacity,
      }
    : {
        ...edge,
        fromPoint: secondGate,
        control1: secondControl,
        control2: firstControl,
        toPoint: firstGate,
        opacity,
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
