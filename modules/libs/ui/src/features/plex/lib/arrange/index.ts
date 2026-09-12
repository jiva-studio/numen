/** The decisions, as values: no DOM, no Vue, no clock. */
export { arrangePlex, type ArrangeInput } from './arrange'
export { measureCrowding, packOptions, type SeatCounts } from './crowding'
export { limitsFor, type Limits, type RoleLimits } from './limits'
export {
  getDropSeat,
  nodeAt,
  resolveDrop,
  seatTowards,
  seatWithoutDirection,
  type Drop,
  type DropInput,
  type DroppedInput,
} from './drop'
export { interpolatePlex } from './interpolate'
export { rowsAndColumns, type Placement, type Seating, type Widths } from './placement'
export { spacingAsSet, spacingFor, type Spacing } from './spacing'
export { routeEdge, routeEdges, routingFor, type Axis, type Routing } from './routing'
export {
  arrowTransformOf,
  pathOf,
  readingPathOf,
  threadOf,
  ARROWHEAD_PATH,
} from './paths'
export { settleTitles } from './titles'
export { clamp01, easeOut, lerp, lerpExtent } from './math'
export {
  DEFAULT_DIRECTION,
  DEFAULT_OPTIONS,
  isVertical,
  resolveOptions,
  type BoxOptions,
  type Direction,
  type LimitOptions,
  type PlexOptions,
  type PlexOptionsInput,
  type RoutingOptions,
  type Size,
} from './options'
