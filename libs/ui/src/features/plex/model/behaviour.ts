/**
 * How the plex behaves under a hand and over time, declared once: how long the
 * attention rests before a box widens, how a node offers to be reached out of,
 * how it is asked for on its own, and the clock a box opens on.
 */
import { EXPANSION_DELAY } from './sizing/expansion'
import { byHandle, type ReachStrategy } from './interaction/reachStrategy'
import { byDoubleClick, type ShowStrategy } from './interaction/destination'
import { browserClock, type Clock } from '@/shared/lib/time/clock'

export interface PlexBehaviour {
  /** How long the attention rests on a box before it widens. Milliseconds. */
  readonly expansionDelay: number
  /** How a node offers to be reached out of. */
  readonly reaching: ReachStrategy
  /** How a node is asked for on its own. */
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
