/** How fast the counts in the corner are moving, as plain values. */

import { readable, tallyOf, type Notice } from './notice'
import { rateOf } from './tally'

/** What one count stood at when it last moved, and when that was. */
export interface Movement {
  readonly done: number
  readonly rate: number
  readonly at: number
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
  at: number,
): ReadonlyMap<string, Movement> => {
  const moving = new Map<string, Movement>()
  for (const notice of readable(notices)) {
    const tally = tallyOf(notice)
    if (tally === undefined) continue
    const before = was.get(notice.id)
    if (before === undefined) {
      moving.set(notice.id, { done: tally.done, rate: 0, at })
      continue
    }
    if (tally.done === before.done) {
      moving.set(notice.id, before)
      continue
    }
    const rate = rateOf(before, tally.done, (at - before.at) / 1000)
    moving.set(notice.id, { done: tally.done, rate, at })
  }
  return moving
}
