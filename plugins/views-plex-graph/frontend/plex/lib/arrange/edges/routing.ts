import {
  getArrow,
  getLength,
  type EdgeCurve,
  type EdgeTitle,
  type LabelType,
  type PlacedEdge,
} from '../../edge'
import type { PlacedNode, Position } from '../../node'
import { cutToFit } from '../../cutToFit'
import { DEFAULT_OPTIONS, isVertical, type PlexOptions, type RoutingOptions } from '../options'
import type { PlexEdge, PlexNodeRelation } from '../../../../types'
import { createTitle, getLabelPlaces, getRise, getShift } from './labels'

/** `auto` means take the axis from the geometry. */
type Axis = 'vertical' | 'horizontal' | 'auto'

export interface EdgeRouting {
  readonly tuning: RoutingOptions
  readonly getAxis: (node: { relation: PlexNodeRelation }) => Axis
  readonly labelWidth?: ((label: string) => number) | undefined
  readonly labelType: LabelType
}

/** The ink of a font nothing measured. */
const NO_TYPE: LabelType = { ascent: 0, descent: 0, xHeight: 0 }

/**
 * The axis is read off the relation, not off the distance between the boxes: the
 * last child in a wide row is further sideways than it is down, and measuring
 * would send its edge out of the focus's side.
 */
export function createRouting(
  options: PlexOptions,
  measureLabel?: (label: string) => number,
  labelType: LabelType = NO_TYPE,
): EdgeRouting {
  return {
    tuning: options.routing,
    labelWidth: measureLabel,
    labelType,
    getAxis: (node) => {
      if (node.relation === 'focus') return 'auto'
      return isVertical(options.direction[node.relation]) ? 'vertical' : 'horizontal'
    },
  }
}

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

function isSeparated(a: PlacedNode, b: PlacedNode, isVertical: boolean): boolean {
  return isVertical
    ? Math.abs(b.y - a.y) > (a.height + b.height) / 2
    : Math.abs(b.x - a.x) > (a.width + b.width) / 2
}

function isVerticalRun(from: PlacedNode, to: PlacedNode, routing: EdgeRouting): boolean {
  const declared = [routing.getAxis(from), routing.getAxis(to)].find((axis) => axis !== 'auto')
  const wanted =
    declared === 'vertical' ||
    (declared === undefined && Math.abs(to.y - from.y) >= Math.abs(to.x - from.x))

  // Two nodes in the same row have no vertical space between their gates.
  // Joining them bottom-to-top there loops down out of one box and back up.
  return isSeparated(from, to, wanted) || !isSeparated(from, to, !wanted) ? wanted : !wanted
}

function getControl(point: Position, vertical: boolean, reach: number): Position {
  return vertical ? { x: point.x, y: point.y + reach } : { x: point.x + reach, y: point.y }
}

function reverse(curve: EdgeCurve): EdgeCurve {
  return {
    fromPoint: curve.toPoint,
    control1: curve.control2,
    control2: curve.control1,
    toPoint: curve.fromPoint,
  }
}

function getRun(a: PlacedNode, b: PlacedNode, isVertical: boolean): number {
  return isVertical
    ? Math.abs(b.y - a.y) - (a.height + b.height) / 2
    : Math.abs(b.x - a.x) - (a.width + b.width) / 2
}

function buildStraightCurve(
  first: PlacedNode,
  second: PlacedNode,
  isVertical: boolean,
  routing: EdgeRouting,
): EdgeCurve {
  const firstGate = getEdgeAnchor(first, isVertical ? 'bottom' : 'right')
  const secondGate = getEdgeAnchor(second, isVertical ? 'top' : 'left')

  const span = isVertical ? secondGate.y - firstGate.y : secondGate.x - firstGate.x
  const { curvature, minReach } = routing.tuning
  const reach = curvature > 0 ? Math.max(minReach, Math.abs(span) * curvature) : 0

  return {
    fromPoint: firstGate,
    control1: getControl(firstGate, isVertical, reach),
    control2: getControl(secondGate, isVertical, -reach),
    toPoint: secondGate,
  }
}

function buildBowCurve(
  first: PlacedNode,
  second: PlacedNode,
  isColumn: boolean,
  routing: EdgeRouting,
): EdgeCurve {
  const middle = isColumn ? (first.x + second.x) / 2 : (first.y + second.y) / 2
  const isNegative = isColumn ? middle < 0 : middle >= 0
  const [positive, negative] = isColumn
    ? (['right', 'left'] as const)
    : (['bottom', 'top'] as const)
  const side = isNegative ? negative : positive
  const firstGate = getEdgeAnchor(first, side)
  const secondGate = getEdgeAnchor(second, side)
  // The bow is minRun deep at the default curvature and flat at 0.
  const { curvature, minRun } = routing.tuning
  const depth = (minRun * curvature) / DEFAULT_OPTIONS.routing.curvature
  const reach = isNegative ? -depth : depth
  const getBowControl = (gate: Position): Position =>
    isColumn ? { x: gate.x + reach, y: gate.y } : { x: gate.x, y: gate.y + reach }
  return {
    fromPoint: firstGate,
    control1: getBowControl(firstGate),
    control2: getBowControl(secondGate),
    toPoint: secondGate,
  }
}

function buildCurve(from: PlacedNode, to: PlacedNode, routing: EdgeRouting): EdgeCurve {
  const isVertical = isVerticalRun(from, to, routing)
  const run = getRun(from, to, isVertical)
  const isBowed =
    from.relation !== 'focus' &&
    to.relation !== 'focus' &&
    run >= 0 &&
    run < routing.tuning.minRun &&
    getRun(from, to, !isVertical) < routing.tuning.minRun

  const fromFirst = isVertical ? from.y <= to.y : from.x <= to.x
  const [first, second] = fromFirst ? [from, to] : [to, from]
  const axisEdge = isBowed
    ? buildBowCurve(first, second, isVertical, routing)
    : buildStraightCurve(first, second, isVertical, routing)

  // The caller's from and to are kept, so the curve may run right to left or
  // bottom to top.
  return fromFirst ? axisEdge : reverse(axisEdge)
}

function createTitles(edge: PlexEdge, curve: EdgeCurve, routing: EdgeRouting): EdgeTitle[] {
  const written = (edge.labels ?? []).map((label, record) => ({ label, record }))
  const kept = written.filter(({ label }) => label.text)
  const labels = kept.map(({ label }) => label)
  const clearance = edge.arrows?.length ? 2 * routing.tuning.arrowClearance : 0
  const room = (getLength(curve) - clearance) / labels.length
  const { labelWidth } = routing
  const places = getLabelPlaces(labels)
  return labels.map((label, index) => {
    const { position, side, heading } = places[index]!
    const { record } = kept[index]!
    return createTitle(
      curve,
      position,
      labelWidth ? cutToFit(label.text, room, labelWidth) : label.text,
      {
        record,
        side,
        shift: getShift(side, routing.labelType),
        rise: getRise(side, routing.labelType),
        ...(heading ? { heading } : {}),
      },
    )
  })
}

interface RouteEdgeOptions {
  readonly routing: EdgeRouting
  readonly opacity?: number
}

/**
 * Route an edge as a cubic curve leaving both boxes square-on. Dropped if
 * either endpoint is not on the canvas.
 */
export function routeEdge(
  edge: PlexEdge,
  byId: ReadonlyMap<string, PlacedNode>,
  routingOrOptions: EdgeRouting | RouteEdgeOptions,
): PlacedEdge | null {
  const from = byId.get(edge.from)
  const to = byId.get(edge.to)
  if (!from || !to || from === to) return null

  const routingOptions = 'tuning' in routingOrOptions ? routingOrOptions : routingOrOptions.routing
  const opacity = 'tuning' in routingOrOptions ? 1 : (routingOrOptions.opacity ?? 1)

  const curve = buildCurve(from, to, routingOptions)

  return {
    ...edge,
    ...curve,
    opacity,
    titles: createTitles(edge, curve, routingOptions),
    arrowheads: edge.arrows?.map((end) => getArrow(curve, end)),
  }
}
