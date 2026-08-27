import type { PlexRelatedSeat } from '../model'
import { SEATS } from '../model'

export type Direction = 'up' | 'down' | 'left' | 'right'

export interface Size {
  readonly width: number
  readonly height: number
}

export interface BoxOptions {
  /** The focus box. Its width is the widest that box is drawn. */
  readonly focusSize: Size
  /**
   * Every other box. Its width is the widest one is drawn, and the width
   * admission is worked out from.
   */
  readonly nodeSize: Size
  /** The narrowest a box is drawn, however little its title needs. */
  readonly minWidth: number
  /**
   * Room kept beside a title for the icon a caller draws there, a gap from it
   * and inside the same padding. A box is drawn this much wider wherever an
   * icon is drawn, the plex having no way to measure one.
   */
  readonly iconWidth: number
  /** One place inside a node, as it is drawn under the box. */
  readonly partHeight: number
  /** How far one level of nesting sets a place in. */
  readonly partIndent: number
  /** Between neighbouring nodes along one line. */
  readonly gap: number
  /** Between one line and the next, further from the focus. */
  readonly lineGap: number
  /** Between the focus box and the first line of any seat. */
  readonly focusGap: number
  /** Kept clear of the window edge, so nothing sits flush against it. */
  readonly margin: number
  /**
   * How far a gap may open beyond its setting when the window has room. This
   * multiple of a gap is the most it is given. At one, a gap never opens.
   */
  readonly spread: number
  /**
   * How far a gap closes below its setting when the window is short of room.
   * This fraction of a gap is the least it is given, and a box narrows towards
   * `minWidth` alongside it. At one, nothing closes.
   */
  readonly squeeze: number
}

export interface LimitOptions {
  /**
   * How many nodes go on one line before a second line is started, when there
   * is no window to measure a line against.
   */
  readonly maxPerLine: number
  /** Lines per seat. Nodes past the last line are reported as overflow. */
  readonly maxLines: number
}

export interface RoutingOptions {
  /** How far an edge holds its leaving direction, as a fraction of the gap. */
  readonly curvature: number
  /** And never less than this, or a short edge sets off crooked. */
  readonly minReach: number
  /**
   * How much of a line an arrowhead takes. A title is set about the middle of
   * its line, so a line carrying one has this much less room at either end.
   */
  readonly arrowRoom: number
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
  readonly direction: Readonly<Record<PlexRelatedSeat, Direction>>
  /**
   * The window the plex is drawn in. Given one, a line runs as long as the
   * window holds and the gaps open into whatever room is left over. Without
   * one, the limits above are taken literally.
   */
  readonly viewport?: Size | undefined
}

export const DEFAULT_DIRECTION: Readonly<Record<PlexRelatedSeat, Direction>> =
  Object.fromEntries(
    Object.entries(SEATS).map(([seat, descriptor]) => [seat, descriptor.grows]),
  ) as Record<PlexRelatedSeat, Direction>

export const DEFAULT_OPTIONS: PlexOptions = {
  focusSize: { width: 176, height: 44 },
  nodeSize: { width: 144, height: 36 },
  minWidth: 72,
  iconWidth: 16,
  partHeight: 22,
  partIndent: 12,
  gap: 16,
  lineGap: 20,
  focusGap: 56,
  margin: 16,
  spread: 2.5,
  squeeze: 0.35,
  maxPerLine: 5,
  maxLines: 4,
  routing: { curvature: 0.55, minReach: 22, arrowRoom: 20 },
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
