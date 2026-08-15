import type { PlexRelatedRole } from '../model'
import { ROLES } from '../model'

export type Direction = 'up' | 'down' | 'left' | 'right'

export interface Size {
  readonly width: number
  readonly height: number
}

export interface BoxOptions {
  readonly focusSize: Size
  readonly nodeSize: Size
  /** Between neighbouring nodes along one line. */
  readonly gap: number
  /** Between one line and the next, further from the focus. */
  readonly lineGap: number
  /** Between the focus box and the first line of any role. */
  readonly focusGap: number
  /** Kept clear of the window edge, so nothing sits flush against it. */
  readonly margin: number
}

export interface LimitOptions {
  /** How many nodes fit on one line before a second line is started. */
  readonly maxPerLine: number
  /** Lines per role. Nodes past the last line are reported as overflow. */
  readonly maxLines: number
}

export interface RoutingOptions {
  /** How far an edge holds its leaving direction, as a fraction of the gap. */
  readonly curvature: number
  /** And never less than this, or a short edge sets off crooked. */
  readonly minReach: number
}

/**
 * The shape of a movement, as fractions of it. Deliberately not mirror
 * images: what is leaving goes early, so the picture is never at its most
 * crowded halfway through.
 */
export interface MotionOptions {
  /** When a node only in the new picture starts to appear. */
  readonly arriveAfter: number
  /** When a node only in the old one has finished going. */
  readonly leaveBefore: number
}

export interface GestureOptions {
  /**
   * How much further sideways than vertical a gesture may go and still count
   * as up or down. Rows are wide and columns narrow, so the wedge that means
   * a parent or a child is wider than a quarter turn — at one, the outermost
   * child of a wide row would be read as something off to the side.
   */
  readonly verticalBias: number
}

export interface PlexOptions extends BoxOptions, LimitOptions {
  readonly routing: RoutingOptions
  readonly motion: MotionOptions
  readonly gesture: GestureOptions
  readonly direction: Readonly<Record<PlexRelatedRole, Direction>>
  /**
   * The window the plex is drawn in. Given one, a row too wide for it wraps
   * sooner and runs deeper rather than reaching past the edge. Without one,
   * the limits above are taken literally.
   */
  readonly viewport?: Size | undefined
}

export const DEFAULT_DIRECTION: Readonly<Record<PlexRelatedRole, Direction>> =
  Object.fromEntries(
    Object.entries(ROLES).map(([role, descriptor]) => [role, descriptor.grows]),
  ) as Record<PlexRelatedRole, Direction>

export const DEFAULT_OPTIONS: PlexOptions = {
  focusSize: { width: 176, height: 44 },
  nodeSize: { width: 144, height: 36 },
  gap: 16,
  lineGap: 20,
  focusGap: 56,
  margin: 16,
  maxPerLine: 5,
  maxLines: 4,
  routing: { curvature: 0.55, minReach: 22 },
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
