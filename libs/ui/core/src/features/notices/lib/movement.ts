/** How fast the counts in the corner are moving, as plain values. */

import { getReadable, findTally, type Notice } from './notice'
import { getRate } from './tally'

/** What one count stood at when it last moved, and when that was. */
export interface Movement {
  readonly completedCount: number
  readonly rate: number
  readonly timestamp: number
}

/**
 * How fast each count is moving, from where it last moved to where it is now.
 *
 * A reading that saw no movement leaves the mark where it is, so the stretch a
 * count stood still over is time the work took and is measured as such.
 *
 * A count that has gone is forgotten, and one that has just arrived is read once
 * before it has a rate.
 */
export const measureMovement = (
  was: ReadonlyMap<string, Movement>,
  notices: readonly Notice[],
  timestamp: number,
): ReadonlyMap<string, Movement> => {
  const movements = new Map<string, Movement>()
  for (const notice of getReadable(notices)) {
    const tally = findTally(notice)
    if (tally === undefined) continue
    const before = was.get(notice.id)
    if (before === undefined) {
      movements.set(notice.id, { completedCount: tally.completedCount, rate: 0, timestamp })
      continue
    }
    if (tally.completedCount === before.completedCount) {
      movements.set(notice.id, before)
      continue
    }
    const rate = getRate(before, tally.completedCount, (timestamp - before.timestamp) / 1000)
    movements.set(notice.id, { completedCount: tally.completedCount, rate, timestamp })
  }
  return movements
}
