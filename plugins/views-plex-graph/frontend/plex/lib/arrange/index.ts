/** The decisions, as values: no DOM, no Vue, no clock. */
export { arrangePlex, type ArrangeInput } from './arrange'
export { measureCrowding, packOptions, type RelationCounts } from './layout/crowding'
export { getLimits, type Limits, type RoleLimits } from './layout/limits'
export {
  getDropRelation,
  findNode,
  resolveDrop,
  getRelationTowards,
  getRelationWithoutDirection,
  type Drop,
  type DropInput,
  type DroppedInput,
} from './edges/drop'
export { interpolatePlex } from './interpolate'
export {
  rowsAndColumns,
  type Placement,
  type NodesByRelation,
  type Widths,
} from './layout/placement'
export { spacingAsSet, findSpacing, type Spacing } from './layout/spacing'
export { routeEdge, routeEdges, createRouting, type Axis, type Routing } from './edges/routing'
export {
  createArrowTransform,
  createPath,
  createReadingPath,
  createThreadPath,
  ARROWHEAD_PATH,
} from './edges/paths'
export { separateTitles } from './edges/titles'
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
