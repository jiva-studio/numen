import { RELATIONS } from '../relation'
import type { Size } from '@numen/ui'
import type { PlexRelation } from '../../../types'

export type Direction = 'up' | 'down' | 'left' | 'right'

export type { Size }

interface BoxOptions {
  readonly focusSize: Size
  readonly nodeSize: Size
  readonly minWidth: number
  readonly iconWidth: number
  readonly headingHeight: number
  readonly headingIndent: number
  readonly gap: number
  readonly lineGap: number
  readonly focusGap: number
  readonly margin: number
  readonly spread: number
  readonly squeeze: number
}

interface LimitOptions {
  readonly maxPerLine: number
  readonly maxLines: number
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

export const DEFAULT_OPTIONS: PlexOptions = {
  focusSize: { width: 176, height: 44 },
  nodeSize: { width: 144, height: 36 },
  minWidth: 72,
  iconWidth: 16,
  headingHeight: 22,
  headingIndent: 12,
  gap: 16,
  lineGap: 20,
  focusGap: 56,
  margin: 16,
  spread: 2.5,
  squeeze: 0.35,
  maxPerLine: 5,
  maxLines: 4,
  maxHeadings: 6,
  routing: { curvature: 0.55, minReach: 22, arrowClearance: 20, minRun: 52 },
  motion: { arriveAfter: 0.35, leaveBefore: 0.45 },
  gesture: { verticalBias: 4 },
  direction: DEFAULT_DIRECTION,
}

/** What a caller may pass: any subset, one level deep. */
export type PlexOptionsInput = Partial<
  Omit<PlexOptions, 'routing' | 'motion' | 'gesture'> & {
    routing: Partial<RoutingOptions>
    motion: Partial<MotionOptions>
    gesture: Partial<GestureOptions>
  }
>

export function resolveOptions(options?: PlexOptionsInput): PlexOptions {
  if (!options) return DEFAULT_OPTIONS
  return {
    ...DEFAULT_OPTIONS,
    ...options,
    routing: { ...DEFAULT_OPTIONS.routing, ...options.routing },
    motion: { ...DEFAULT_OPTIONS.motion, ...options.motion },
    gesture: { ...DEFAULT_OPTIONS.gesture, ...options.gesture },
    direction: { ...DEFAULT_DIRECTION, ...options.direction },
  }
}

export const isVertical = (direction: Direction): boolean =>
  direction === 'up' || direction === 'down'
