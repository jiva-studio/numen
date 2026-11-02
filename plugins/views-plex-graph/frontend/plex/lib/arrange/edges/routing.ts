import {
  getArrow,
  getHeading,
  getLength,
  type EdgeCurve,
  type EdgeHeading,
  type EdgeLabel,
  type EdgeSide,
  type EdgeTitle,
  type PlacedEdge,
  type PlexEdge,
} from '../../edge'
import type { PlacedNode, Position } from '../../node'
import type { PlexNodeRelation } from '../../relation'
import { isVertical, type PlexOptions, type RoutingOptions } from '../options'

/** `auto` means take the axis from the geometry. */
export type Axis = 'vertical' | 'horizontal' | 'auto'

export interface Routing extends RoutingOptions {
  readonly getAxis: (node: { relation: PlexNodeRelation }) => Axis
  /**
   * How wide a title is set. Text is measured where the plex is drawn; here it
   * arrives as a number, and without a measurer the whole label is drawn.
   */
  readonly labelWidth?: ((label: string) => number) | undefined
  /**
   * The height of one line of a title, perpendicular to the line it is set on.
   * It is measured in the title's own type, and is zero when nothing measured
   * it.
   */
  readonly labelDepth: number
}

/**
 * The axis is read off the relation, not off the distance between the boxes: the
 * last child in a wide row is further sideways than it is down, and measuring
 * would send its edge out of the focus's side.
 */
export function createRouting(
  options: PlexOptions,
  measureLabel?: (label: string) => number,
  labelDepth = 0,
): Routing {
  return {
    ...options.routing,
    labelWidth: measureLabel,
    labelDepth,
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
 * is the record of the `from` note. One label is centred. Several are spread
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
 * How far from the line the centre of a title is, for letters this deep: nothing on the line, else half of
 * their depth and the gap, so the near edge of the letters is `LABEL_GAP` from the line.
 */
export const getShift = (side: EdgeSide, depth: number): number => side * (depth / 2 + LABEL_GAP)

/**
 * Where and how each label of one line is set, in the order the labels are written.
 * One label is centred along the line and on it, its vertical middle on the line, read the readable way along it.
 * Two labels are set near the two ends, on opposite sides, each read in the direction of its
 * record: when both records have a head, each label is near the end its head is at and reads toward it;
 * otherwise each is near the note that wrote it and reads away from it, the first on the left of
 * the line taken from `from`, the second on the right.
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
  const { anchor = position, side, shift } = title
  const heading = title.heading ?? getHeading(curve, position)
  return {
    anchor,
    heading,
    side,
    shift,
    words,
    wordsAt: heading === 'against' ? 1 - position : position,
  }
}

/** What a title keeps wherever it is moved to along its line. */
export interface TitleSetting {
  /** Where the title was asked to be set; the position it is created at when omitted. */
  readonly anchor?: number
  /** The way the words are read where the record fixes it; the way the line runs, readable, when omitted. */
  readonly heading?: EdgeHeading
  readonly side: EdgeSide
  readonly shift: number
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

/** Whether an edge runs vertically or horizontally. */
function isVerticalRun(from: PlacedNode, to: PlacedNode, routing: Routing): boolean {
  const declared = [routing.getAxis(from), routing.getAxis(to)].find((axis) => axis !== 'auto')
  const wanted =
    declared === 'vertical' ||
    (declared === undefined && Math.abs(to.y - from.y) >= Math.abs(to.x - from.x))

  // Two nodes in the same row have no vertical space between their gates.
  // Joining them bottom-to-top there loops down out of one box and back up.
  return isSeparated(from, to, wanted) || !isSeparated(from, to, !wanted) ? wanted : !wanted
}

/** The control point standing out from a getEdgeAnchor along the run. */
function getControl(point: Position, vertical: boolean, reach: number): Position {
  return vertical ? { x: point.x, y: point.y + reach } : { x: point.x + reach, y: point.y }
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

/** The clear space between two boxes along an axis; negative where they overlap on it. */
function getRun(a: PlacedNode, b: PlacedNode, isVertical: boolean): number {
  return isVertical
    ? Math.abs(b.y - a.y) - (a.height + b.height) / 2
    : Math.abs(b.x - a.x) - (a.width + b.width) / 2
}

/** A cubic leaving both boxes square-on along the axis, read from the first box. */
function straightBetween(
  first: PlacedNode,
  second: PlacedNode,
  isVertical: boolean,
  routing: Routing,
): EdgeCurve {
  const firstGate = getEdgeAnchor(first, isVertical ? 'bottom' : 'right')
  const secondGate = getEdgeAnchor(second, isVertical ? 'top' : 'left')

  const span = isVertical ? secondGate.y - firstGate.y : secondGate.x - firstGate.x
  const reach = Math.max(routing.minReach, Math.abs(span) * routing.curvature)

  return {
    fromPoint: firstGate,
    control1: getControl(firstGate, isVertical, reach),
    control2: getControl(secondGate, isVertical, -reach),
    toPoint: secondGate,
  }
}

/**
 * A line from one side of both boxes that bows out by `minRun`, for boxes with
 * no room between them for an arrowhead and a word. A row bows towards the focus,
 * into the space between the rows, and a column bows away from it.
 */
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
  const reach = isNegative ? -routing.minRun : routing.minRun
  const getBowControl = (gate: Position): Position =>
    isColumn ? { x: gate.x + reach, y: gate.y } : { x: gate.x, y: gate.y + reach }
  return {
    fromPoint: firstGate,
    control1: getBowControl(firstGate),
    control2: getBowControl(secondGate),
    toPoint: secondGate,
  }
}

/**
 * A cubic leaving both boxes square-on, read from the caller's end.
 *
 * Two boxes side by side in a row or one above the other in a column, with
 * less than `minRun` between them and no room along the other axis, are joined
 * by a bow out of their sides. Edges at the focus keep the space the focus is given.
 */
function curveBetween(from: PlacedNode, to: PlacedNode, routing: Routing): EdgeCurve {
  const isVertical = isVerticalRun(from, to, routing)
  const run = getRun(from, to, isVertical)
  const isBowed =
    from.relation !== 'focus' &&
    to.relation !== 'focus' &&
    run >= 0 &&
    run < routing.minRun &&
    getRun(from, to, !isVertical) < routing.minRun

  const fromFirst = isVertical ? from.y <= to.y : from.x <= to.x
  const [first, second] = fromFirst ? [from, to] : [to, from]
  const axisEdge = isBowed
    ? bowBetween(first, second, isVertical, routing)
    : straightBetween(first, second, isVertical, routing)

  // The caller's from and to are kept, so the curve may run right to left or
  // bottom to top.
  return fromFirst ? axisEdge : reverse(axisEdge)
}

/**
 * The titles a curve carries. The labels of one line share its length, less the
 * room kept for the arrowheads.
 */
function createTitles(edge: PlexEdge, curve: EdgeCurve, routing: Routing): EdgeTitle[] {
  const labels = (edge.labels ?? []).filter((label) => label.text)
  const room =
    (getLength(curve) - (edge.arrows?.length ? 2 * routing.arrowClearance : 0)) / labels.length
  const { labelWidth } = routing
  const places = getLabelPlaces(labels)
  return labels.map((label, index) => {
    const { position, side, heading } = places[index]!
    return createTitle(
      curve,
      position,
      labelWidth ? cutToFit(label.text, room, labelWidth) : label.text,
      { side, shift: getShift(side, routing.labelDepth), ...(heading ? { heading } : {}) },
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

  const routingOptions =
    'curvature' in routingOrOptions ? routingOrOptions : routingOrOptions.routing
  const opacity = 'curvature' in routingOrOptions ? 1 : (routingOrOptions.opacity ?? 1)

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
  const routingOptions =
    'curvature' in routingOrOptions ? routingOrOptions : routingOrOptions.routing
  const getOpacity =
    'curvature' in routingOrOptions ? () => 1 : (routingOrOptions.opacity ?? (() => 1))

  return edges
    .map((edge) => routeEdge(edge, byId, { routing: routingOptions, opacity: getOpacity(edge) }))
    .filter((edge): edge is PlacedEdge => edge !== null)
}
