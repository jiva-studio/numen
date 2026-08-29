/**
 * How a node offers to be reached out of.
 *
 * A hand finds a handle by moving over the node and presses it. A finger has no
 * hover, so there is no handle under it to find; it rests on the node instead,
 * and the gesture carries on from where it rested.
 *
 * Which of the two a plex offers is the caller's, so neither is written into
 * the node.
 */
import { useHold, HOLD } from './holding'

/** The node a strategy is watching, in the only two terms it needs. */
export interface ReachingSite {
  /** Whether this node will take a gesture now. */
  readonly ready: () => boolean
  /** Begin one from this press, which the gesture then follows. */
  readonly reach: (event: PointerEvent) => void
}

export interface Reaching {
  /** Whether the node draws a handle for a pointer to press. */
  readonly handle: boolean
  /** What the node listens for besides. Called once, inside the node's scope. */
  readonly listeners: (site: ReachingSite) => Record<string, (event: PointerEvent) => void>
}

/** The handle, under the hand and under the keyboard. */
export const byHandle: Reaching = {
  handle: true,
  listeners: () => ({}),
}

/** A finger left still on the node. Milliseconds, if the wait is to be another. */
export const byHolding = (after: number = HOLD): Reaching => ({
  handle: false,
  listeners: (site) => {
    const held = useHold(site.ready, site.reach, () => after)
    return {
      pointerdown: held.down,
      pointermove: held.move,
      pointerup: held.letGo,
      pointercancel: held.letGo,
    }
  },
})
