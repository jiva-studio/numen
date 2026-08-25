import {
  headingOf,
  lengthOf,
  type EdgeCurve,
  type PlacedEdge,
  type PlacedNode,
  type PlexEdge,
  type PlexSeat,
  type Point,
} from '../model'
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
}

/**
 * The axis is read off the seat, not off the distance between the boxes: the
 * last child in a wide row is further sideways than it is down, and measuring
 * would send its edge out of the focus's side.
 */
export function routingFor(
  options: PlexOptions,
  measureLabel?: (label: string) => number,
): Routing {
  return {
    ...options.routing,
    labelWidth: measureLabel,
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

/**
 * The words a curve has room for: the longest start of the label that fits it,
 * the ellipsis included. The prefix is found by halving, and a curve with room
 * for nothing carries the ellipsis alone.
 */
function cutToFit(
  label: string,
  room: number,
  width: (label: string) => number,
): string {
  if (width(label) <= room) return label

  const letters = [...label]
  const ended = (count: number) =>
    `${letters.slice(0, count).join('').trimEnd()}${ELLIPSIS}`

  let fits = 0
  let over = letters.length
  while (fits + 1 < over) {
    const middle = Math.floor((fits + over) / 2)
    if (width(ended(middle)) <= room) fits = middle
    else over = middle
  }
  return ended(fits)
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

  // The caller's from and to are kept, so the curve may run right to left or
  // bottom to top.
  const curve: EdgeCurve = fromFirst
    ? {
        fromPoint: firstGate,
        control1: firstControl,
        control2: secondControl,
        toPoint: secondGate,
      }
    : {
        fromPoint: secondGate,
        control1: secondControl,
        control2: firstControl,
        toPoint: firstGate,
      }

  const words =
    edge.label !== undefined && routing.labelWidth
      ? cutToFit(edge.label, lengthOf(curve), routing.labelWidth)
      : edge.label

  return { ...edge, ...curve, opacity, heading: headingOf(curve), words }
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
