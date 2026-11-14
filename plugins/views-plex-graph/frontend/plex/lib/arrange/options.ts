import { RELATIONS } from '../relation'
import { getFocusSize, getNodeSize, type Padding } from './look'
import type { Size } from '@numen/ui'
import type { PlexRelation } from '../../../types'

export type Direction = 'up' | 'down' | 'left' | 'right'

export type { Size }

export type { Padding }

/**
 * `focusScale` is how much larger the focus pill is than the box its title would have as a
 * neighbor. `room` is empty space kept on every side of a box, on top of the gaps.
 */
interface BoxOptions {
  readonly focusSize: Size
  readonly nodeSize: Size
  readonly focusScale: number
  readonly padding: Padding
  readonly room: number
  readonly minWidth: number
  readonly iconWidth: number
  readonly headingHeight: number
  readonly headingIndent: number
  readonly gap: number
  readonly lineGap: number
  readonly focusGap: number
  readonly margin: number
  readonly squeeze: number
}

interface LimitOptions {
  readonly maxPerCluster: number
  readonly maxHeadings: number
}

export interface RoutingOptions {
  readonly curvature: number
  readonly minReach: number
  readonly arrowClearance: number
  readonly minRun: number
}

/**
 * The shape of a movement, as fractions of it. The two directions are
 * deliberately asymmetric: the departing element moves early, so the layout is
 * never at its most crowded halfway through.
 */
interface MotionOptions {
  readonly arriveAfter: number
  readonly leaveBefore: number
}

interface GestureOptions {
  readonly verticalBias: number
}

export interface PlexOptions extends BoxOptions, LimitOptions {
  readonly routing: RoutingOptions
  readonly motion: MotionOptions
  readonly gesture: GestureOptions
  readonly direction: Readonly<Record<PlexRelation, Direction>>
  readonly viewport?: Size | undefined
}

const DEFAULT_DIRECTION: Readonly<Record<PlexRelation, Direction>> = Object.fromEntries(
  Object.entries(RELATIONS).map(([relation, descriptor]) => [relation, descriptor.grows]),
) as Record<PlexRelation, Direction>

const DEFAULT_PADDING: Padding = { inline: 10, block: 9.25 }
const DEFAULT_NODE_SIZE = getNodeSize(144, DEFAULT_PADDING)
const DEFAULT_FOCUS_SCALE = 1.22

export const DEFAULT_OPTIONS: PlexOptions = {
  focusSize: getFocusSize(DEFAULT_NODE_SIZE, DEFAULT_FOCUS_SCALE),
  nodeSize: DEFAULT_NODE_SIZE,
  focusScale: DEFAULT_FOCUS_SCALE,
  padding: DEFAULT_PADDING,
  room: 0,
  minWidth: 72,
  iconWidth: 16,
  headingHeight: 22,
  headingIndent: 12,
  gap: 16,
  lineGap: 20,
  focusGap: 56,
  margin: 16,
  squeeze: 0.35,
  maxPerCluster: 20,
  maxHeadings: 6,
  routing: { curvature: 0.55, minReach: 22, arrowClearance: 20, minRun: 52 },
  motion: { arriveAfter: 0.35, leaveBefore: 0.45 },
  gesture: { verticalBias: 4 },
  direction: DEFAULT_DIRECTION,
}

/**
 * What a caller may pass: any subset, one level deep. A box size left out is worked out
 * from the padding and the focus scale; one given is kept as it is, so resolving options
 * that were resolved and scaled before gives them back unchanged.
 */
export type PlexOptionsInput = Partial<
  Omit<PlexOptions, 'routing' | 'motion' | 'gesture' | 'padding'> & {
    padding: Partial<Padding>
    routing: Partial<RoutingOptions>
    motion: Partial<MotionOptions>
    gesture: Partial<GestureOptions>
  }
>

export function resolveOptions(options?: PlexOptionsInput): PlexOptions {
  if (!options) return DEFAULT_OPTIONS
  const padding = { ...DEFAULT_PADDING, ...options.padding }
  const nodeSize = options.nodeSize ?? getNodeSize(DEFAULT_NODE_SIZE.width, padding)
  const focusScale = options.focusScale ?? DEFAULT_FOCUS_SCALE
  return {
    ...DEFAULT_OPTIONS,
    ...options,
    padding,
    nodeSize,
    focusScale,
    focusSize: options.focusSize ?? getFocusSize(nodeSize, focusScale),
    routing: { ...DEFAULT_OPTIONS.routing, ...options.routing },
    motion: { ...DEFAULT_OPTIONS.motion, ...options.motion },
    gesture: { ...DEFAULT_OPTIONS.gesture, ...options.gesture },
    direction: { ...DEFAULT_DIRECTION, ...options.direction },
  }
}

export const isVertical = (direction: Direction): boolean =>
  direction === 'up' || direction === 'down'
