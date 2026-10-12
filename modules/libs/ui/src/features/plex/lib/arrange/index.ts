/** The decisions, as values: no DOM, no Vue, no clock. */
export { arrangePlex, type ArrangeInput } from './arrange'
export { measureCrowding, packOptions, type RelationCounts } from './crowding'
export { getLimits, type Limits, type RoleLimits } from './limits'
export {
  getDropRelation,
  findNode,
  resolveDrop,
  getRelationTowards,
  getRelationWithoutDirection,
  type Drop,
  type DropInput,
  type DroppedInput,
} from './drop'
export { interpolatePlex } from './interpolate'
export { rowsAndColumns, type Placement, type NodesByRelation, type Widths } from './placement'
export { spacingAsSet, findSpacing, type Spacing } from './spacing'
export { routeEdge, routeEdges, createRouting, type Axis, type Routing } from './routing'
export {
  createArrowTransform,
  createPath,
  createReadingPath,
  createThreadPath,
  ARROWHEAD_PATH,
} from './paths'
export { separateTitles } from './titles'
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
