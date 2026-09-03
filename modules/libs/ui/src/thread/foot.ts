/**
 * The foot of an area that scrolls: where it is, and whether it is in view.
 *
 * No DOM and no clock, only the three numbers an area is read by.
 */

/** What the foot is worked out from, in pixels. */
export interface Scrolls {
  /** How far down from the head the area stands. */
  readonly scrollTop: number
  /** How tall everything in the area is. */
  readonly scrollHeight: number
  /** How tall the part of it that shows is. */
  readonly clientHeight: number
}

/** How near the foot still counts as being at it, in pixels. */
export const SLACK = 16

/** How far down an area stands with its foot in view. */
export const footOf = ({ scrollHeight, clientHeight }: Scrolls): number =>
  Math.max(0, scrollHeight - clientHeight)

/** Whether the foot is in view, within the slack. */
export const atFoot = (area: Scrolls, slack = SLACK): boolean =>
  footOf(area) - area.scrollTop <= slack
