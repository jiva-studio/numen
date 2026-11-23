/** The decisions, as values: no DOM, no Vue, no clock. */
export { arrangePlex, type ArrangeInput } from './arrange'

export { getDropRelation, resolveDrop, getRelationWithoutDirection, type Drop } from './edges/drop'
export { interpolatePlex } from './interpolate'
export { NO_ROOM, type NodeRoom, type Placement } from './layout/placement'

export {
  createArrowTransform,
  createPath,
  createReadingPath,
  createThreadPath,
  ARROWHEAD_PATH,
} from './edges/paths'

export { computeEaseOut as easeOut, interpolate as lerp } from './math'
export {
  DEFAULT_OPTIONS,
  resolveOptions,
  type PlexOptions,
  type PlexOptionsInput,
  type Size,
} from './options'
