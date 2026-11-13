/**
 * How the plex responds to pointer input and to time, declared once: the delay
 * before a box widens, how a node offers to be reached, how a node is requested
 * on its own, and the clock a box opens on.
 */
import { EXPANSION_DELAY } from './sizing/expansion'
import { byHandle, type ReachStrategy } from './interaction/reachStrategy'
import { byDoubleClick, type ShowStrategy } from './interaction/destination'
import { browserClock, type Clock } from '@numen/ui'

export interface PlexBehavior {
  readonly expansionDelay: number
  readonly reaching: ReachStrategy
  readonly showing: ShowStrategy
  readonly clock: Clock
}

/** The handle, the second click, the browser's clock, and the usual wait. */
export const DEFAULT_BEHAVIOR: PlexBehavior = {
  expansionDelay: EXPANSION_DELAY,
  reaching: byHandle,
  showing: byDoubleClick,
  clock: browserClock,
}

export const getBehavior = (given: PlexBehavior): PlexBehavior => ({
  expansionDelay: given.expansionDelay,
  reaching: given.reaching,
  showing: given.showing,
  clock: given.clock,
})
