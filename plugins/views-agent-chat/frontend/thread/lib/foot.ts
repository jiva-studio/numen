/**
 * The foot of an area that scrolls: where it is, and whether it is in view.
 *
 * No DOM and no clock, only the three numbers an area is read by.
 */
import { SLACK } from './slack'

/** What the foot is worked out from, in pixels. */
export interface ScrollMetrics {
  readonly scrollTop: number
  readonly scrollHeight: number
  readonly clientHeight: number
}

/** The scroll offset of an area with its foot in view. */
export const getFoot = ({ scrollHeight, clientHeight }: ScrollMetrics): number =>
  Math.max(0, scrollHeight - clientHeight)

/** Whether the foot is in view, within the slack. */
export const isAtFoot = (area: ScrollMetrics, slack = SLACK): boolean =>
  getFoot(area) - area.scrollTop <= slack
