/** The decisions, as values: no DOM, no Vue, no clock. */
export { arrangePlex, type ArrangeInput } from './arrange'
export { limitsFor, type Limits, type RoleLimits } from './limits'
export { interpolatePlex } from './interpolate'
export { rowsAndColumns, type Placement, type Seating } from './placement'
export { routeEdge, routeEdges, routingFor, type Axis, type Routing } from './routing'
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
