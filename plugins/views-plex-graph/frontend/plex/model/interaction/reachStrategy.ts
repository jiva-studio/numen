/**
 * How a node offers a gesture start. A mouse finds a handle by hovering over
 * the node and presses it; touch has no hover, so the press rests on the node
 * and the gesture continues from there.
 *
 * The caller of the plex chooses which of the two applies, so neither is
 * hard-coded in the node.
 */
import { useLongPress, LONG_PRESS_DELAY } from './longPress'

/** The node a strategy is watching, in the only two terms it needs. */
export interface ReachSite {
  /** Whether this node will take a gesture now. */
  readonly ready: () => boolean
  /** Begin one from this press, which the gesture then follows. */
  readonly reach: (event: PointerEvent) => void
}

export interface ReachStrategy {
  /** Whether the node draws a handle for a pointer to press. */
  readonly hasHandle: boolean
  /** The additional listeners the node attaches. Called once, inside the node's scope. */
  readonly listeners: (site: ReachSite) => Record<string, (event: PointerEvent) => void>
}

/** The handle strategy, for pointer and keyboard. */
export const byHandle: ReachStrategy = {
  hasHandle: true,
  listeners: () => ({}),
}

/** A finger left still on the node. Milliseconds, if the wait is to be another. */
export const byLongPress = (after: number = LONG_PRESS_DELAY): ReachStrategy => ({
  hasHandle: false,
  listeners: (site) => {
    const target = useLongPress(site.ready, site.reach, () => after)
    return {
      pointerdown: target.onPointerDown,
      pointermove: target.onPointerMove,
      pointerup: target.letGo,
      pointercancel: target.letGo,
    }
  },
})
