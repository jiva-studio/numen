import {
  getArrow,
  getHeading,
  getLength,
  type EdgeCurve,
  type EdgeHeading,
  type EdgeSide,
  type EdgeTitle,
  type LabelType,
  type PlacedEdge,
} from '../../edge'
import type { PlacedNode, Position } from '../../node'
import { isVertical, type PlexOptions, type RoutingOptions } from '../options'
import type { EdgeLabel, PlexEdge, PlexNodeRelation } from '../../../../types'

/** `auto` means take the axis from the geometry. */
type Axis = 'vertical' | 'horizontal' | 'auto'

export interface Routing {
  readonly tuning: RoutingOptions
  readonly getAxis: (node: { relation: PlexNodeRelation }) => Axis
  readonly labelWidth?: ((label: string) => number) | undefined
  readonly labelType: LabelType
}

/** The ink of a font nothing measured. */
const NO_TYPE: LabelType = { ascent: 0, descent: 0, xHeight: 0 }

/** The depth of one line of a title across its line: from the top of a capital to the bottom of a descender. */
export const getDepth = (type: LabelType): number => type.ascent + type.descent

/**
 * The axis is read off the relation, not off the distance between the boxes: the
 * last child in a wide row is further sideways than it is down, and measuring
 * would send its edge out of the focus's side.
 */
export function createRouting(
  options: PlexOptions,
  measureLabel?: (label: string) => number,
  labelType: LabelType = NO_TYPE,
): Routing {
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

/** The one character a cut title ends in. */
const ELLIPSIS = '…'

/** The middle of a line, where a title with no end is set until something is in the way. */
export const MIDDLE = 0.5

/** How far from its end a title placed near an end is set, as a fraction of the line. */
const NEAR_END = 0.2

/**
 * Where each of `count` labels of one line is set, as a fraction of the line's
 * length from its `from` end, in the order the labels are written: the first
 * is the record of the `from` note. One label is centered. Several are spread
 * from near the start to near the end, so each reads as belonging to its own end.
 */
export const getLabelPositions = (count: number): number[] => {
  if (count <= 1) return [MIDDLE]
  return Array.from(
    { length: count },
    (_, index) => NEAR_END + ((1 - 2 * NEAR_END) * index) / (count - 1),
  )
}

/**
 * The gap, in plex units, between the line and the near edge of the letters of
 * a label of a pair: as close as the letters go without touching the line.
 * It is the same for every kind of line and for both labels of a pair.
 */
export const LABEL_GAP = 2

/**
 * How far from the line the center of a title is: nothing for a title on the line, whose
 * lowercase letters are centered on it; else half of the depth of the ink and the gap, so the
 * near edge of the letters is `LABEL_GAP` from the line.
 */
export const getShift = (side: EdgeSide, type: LabelType): number =>
  side * (getDepth(type) / 2 + LABEL_GAP)

/**
 * How far below the center of a title its baseline is, across the line. The center of a title
 * on the line is the middle of the x-height band, of a title beside it the middle of the ink
 * from capital to descender. The baseline is set by this alone, so no engine's own
 * idea of a middle decides where the letters stand.
 */
export const getRise = (side: EdgeSide, type: LabelType): number =>
  side === 0 ? type.xHeight / 2 : (type.ascent - type.descent) / 2

/**
 * Where and how each label of one line is set, in the order the labels are written.
 * One label is centered along the line and on it, its vertical middle on the line, read the readable way along it.
 * Two labels are set near the two ends, on opposite sides. When both records have a head, each label is near the end its head is at and reads toward it;
 * otherwise each is near the note that wrote it and reads away from it, the first on the left of the line taken from `from`, the second on the right.
 */
export function getLabelPlaces(labels: readonly EdgeLabel[]): LabelPlace[] {
  if (labels.length <= 1) return labels.map(() => ({ position: MIDDLE, side: 0 }))
  const positions = getLabelPositions(labels.length)
  const isHeaded = labels.every((label) => label.end !== undefined)
  return labels.map((label, index) => {
    const isToward = isHeaded ? label.end === 'to' : index === 0
    const position = isHeaded ? (isToward ? positions.at(-1)! : positions[0]!) : positions[index]!
    const side: EdgeSide = isToward ? -1 : 1
    return { position, side, heading: isToward ? 'along' : 'against' }
  })
}

/** Where a label is set: its place along the line, its side, and the way it is read where its record fixes that. */
export interface LabelPlace {
  readonly position: number
  readonly side: EdgeSide
  readonly heading?: EdgeHeading
}

/**
 * The title set at a fraction of the curve from its `from` end, measured along
 * the curve the way its words are read.
 */
export function createTitle(
  curve: EdgeCurve,
  ...[position, words, title]: [position: number, words: string, title: TitleSetting]
): EdgeTitle {
  const { anchor = position, side, shift, rise } = title
  const heading = title.heading ?? getHeading(curve, position)
  return {
    anchor,
    heading,
    side,
    shift,
    rise,
    words,
    wordsAt: heading === 'against' ? 1 - position : position,
  }
}

/** What a title keeps wherever it is moved to along its line. */
export interface TitleSetting {
  readonly anchor?: number
  readonly heading?: EdgeHeading
  readonly side: EdgeSide
  readonly shift: number
  readonly rise: number
}

/**
 * The longest prefix of the label that fits the curve, including the ellipsis.
 * The prefix is found by bisection, and a curve with no space for any text
 * returns the ellipsis alone.
 */
export function cutToFit(label: string, room: number, width: (label: string) => number): string {
  if (width(label) <= room) return label

  const letters = [...label]
  const truncateTo = (count: number) => `${letters.slice(0, count).join('').trimEnd()}${ELLIPSIS}`

  let fits = 0
  let overCount = letters.length
  while (fits + 1 < overCount) {
    const middle = Math.floor((fits + overCount) / 2)
    if (width(truncateTo(middle)) <= room) fits = middle
    else overCount = middle
  }
  return truncateTo(fits)
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

function isVerticalRun(from: PlacedNode, to: PlacedNode, routing: Routing): boolean {
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

function straightBetween(
  first: PlacedNode,
  second: PlacedNode,
  isVertical: boolean,
  routing: Routing,
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

function bowBetween(
  first: PlacedNode,
  second: PlacedNode,
  isColumn: boolean,
  routing: Routing,
): EdgeCurve {
  const middle = isColumn ? (first.x + second.x) / 2 : (first.y + second.y) / 2
  const isNegative = isColumn ? middle < 0 : middle >= 0
  const [positive, negative] = isColumn
    ? (['right', 'left'] as const)
    : (['bottom', 'top'] as const)
  const side = isNegative ? negative : positive
  const firstGate = getEdgeAnchor(first, side)
  const secondGate = getEdgeAnchor(second, side)
  const reach = isNegative ? -routing.tuning.minRun : routing.tuning.minRun
  const getBowControl = (gate: Position): Position =>
    isColumn ? { x: gate.x + reach, y: gate.y } : { x: gate.x, y: gate.y + reach }
  return {
    fromPoint: firstGate,
    control1: getBowControl(firstGate),
    control2: getBowControl(secondGate),
    toPoint: secondGate,
  }
}

function curveBetween(from: PlacedNode, to: PlacedNode, routing: Routing): EdgeCurve {
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
    ? bowBetween(first, second, isVertical, routing)
    : straightBetween(first, second, isVertical, routing)

  // The caller's from and to are kept, so the curve may run right to left or
  // bottom to top.
  return fromFirst ? axisEdge : reverse(axisEdge)
}

function createTitles(edge: PlexEdge, curve: EdgeCurve, routing: Routing): EdgeTitle[] {
  const labels = (edge.labels ?? []).filter((label) => label.text)
  const clearance = edge.arrows?.length ? 2 * routing.tuning.arrowClearance : 0
  const room = (getLength(curve) - clearance) / labels.length
  const { labelWidth } = routing
  const places = getLabelPlaces(labels)
  return labels.map((label, index) => {
    const { position, side, heading } = places[index]!
    return createTitle(
      curve,
      position,
      labelWidth ? cutToFit(label.text, room, labelWidth) : label.text,
      {
        side,
        shift: getShift(side, routing.labelType),
        rise: getRise(side, routing.labelType),
        ...(heading ? { heading } : {}),
      },
    )
  })
}

export interface RouteEdgeOptions {
  readonly routing: Routing
  readonly opacity?: number
}

/**
 * Route an edge as a cubic curve leaving both boxes square-on. Dropped if
 * either endpoint is not on the canvas.
 */
export function routeEdge(
  edge: PlexEdge,
  byId: ReadonlyMap<string, PlacedNode>,
  routingOrOptions: Routing | RouteEdgeOptions,
): PlacedEdge | null {
  const from = byId.get(edge.from)
  const to = byId.get(edge.to)
  if (!from || !to || from === to) return null

  const routingOptions = 'tuning' in routingOrOptions ? routingOrOptions : routingOrOptions.routing
  const opacity = 'tuning' in routingOrOptions ? 1 : (routingOrOptions.opacity ?? 1)

  const curve = curveBetween(from, to, routingOptions)

  return {
    ...edge,
    ...curve,
    opacity,
    titles: createTitles(edge, curve, routingOptions),
    arrowheads: edge.arrows?.map((end) => getArrow(curve, end)),
  }
}

export interface RouteEdgesOptions {
  readonly routing: Routing
  readonly opacity?: (edge: PlexEdge) => number
}

export function routeEdges(
  edges: readonly PlexEdge[],
  byId: ReadonlyMap<string, PlacedNode>,
  routingOrOptions: Routing | RouteEdgesOptions,
): PlacedEdge[] {
  const routingOptions = 'tuning' in routingOrOptions ? routingOrOptions : routingOrOptions.routing
  const getOpacity =
    'tuning' in routingOrOptions ? () => 1 : (routingOrOptions.opacity ?? (() => 1))

  return edges
    .map((edge) => routeEdge(edge, byId, { routing: routingOptions, opacity: getOpacity(edge) }))
    .filter((edge): edge is PlacedEdge => edge !== null)
}
