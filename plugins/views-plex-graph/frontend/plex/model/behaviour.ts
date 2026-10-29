/**
 * How the plex responds to pointer input and to time, declared once: the delay
 * before a box widens, how a node offers to be reached, how a node is requested
 * on its own, and the clock a box opens on.
 */
import { EXPANSION_DELAY } from './sizing/expansion'
import { byHandle, type ReachStrategy } from './interaction/reachStrategy'
import { byDoubleClick, type ShowStrategy } from './interaction/destination'
import { browserClock, type Clock } from '@numen/ui'

export interface PlexBehaviour {
  /** How long the attention rests on a box before it widens. Milliseconds. */
  readonly expansionDelay: number
  /** How a node offers to be reached out of. */
  readonly reaching: ReachStrategy
  /** How a node is requested on its own. */
  readonly showing: ShowStrategy
  /** The clock a box opens on. */
  readonly clock: Clock
}

/** The handle, the second click, the browser's clock, and the usual wait. */
export const DEFAULT_BEHAVIOUR: PlexBehaviour = {
  expansionDelay: EXPANSION_DELAY,
  reaching: byHandle,
  showing: byDoubleClick,
  clock: browserClock,
}
