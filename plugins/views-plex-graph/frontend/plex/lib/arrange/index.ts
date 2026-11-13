/** The decisions, as values: no DOM, no Vue, no clock. */
export { arrangePlex, type ArrangeInput } from './arrange'

export { getDropRelation, resolveDrop, getRelationWithoutDirection, type Drop } from './edges/drop'
export { interpolatePlex } from './interpolate'
export { clusters } from './layout/clusters'
export { NO_ROOM, type NodeRoom, type Placement, type Rooms } from './layout/placement'

export { createTitle, getLabelPlaces, getRise, getShift, LABEL_GAP } from './edges/routing'
export {
  createArrowTransform,
  createPath,
  createReadingPath,
  createThreadPath,
  ARROWHEAD_PATH,
} from './edges/paths'

export { clamp01, easeOut, lerp } from './math'
export {
  DEFAULT_OPTIONS,
  resolveOptions,
  type PlexOptions,
  type PlexOptionsInput,
  type Size,
} from './options'
