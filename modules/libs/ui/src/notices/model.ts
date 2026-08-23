/**
 * What a notice is, as plain values. No DOM, no clock, no measurement.
 */

import { rateOf, type Counting } from '../activity/model'

/**
 * One thing running behind the window.
 *
 * `done` and `total` are the work in hand, not the size of what it is being
 * done to: somebody who changed one note is waiting on one thing.
 */
export interface Notice {
  /** Whatever the caller addresses this notice by. Never read, only handed back. */
  readonly id: string
  /** What is happening, in the words it is to be shown by. */
  readonly says: string
  /** What it is happening to, when that is worth saying. */
  readonly about?: string
  /** Where it has got to, when there is a total to count against. */
  readonly done?: number
  readonly total?: number
  /** What that count counts. */
  readonly counting?: Counting
  /** Whether it is running now, or is a fact that is simply so. */
  readonly working?: boolean
  /** Why it stopped, when it stopped badly. It is drawn as trouble. */
  readonly trouble?: string
  /**
   * Whether a person asked for this and is waiting to be told it began. One of
   * these is drawn the moment it arrives.
   */
  readonly asked?: boolean
}

/**
 * The notices worth drawing: the ones that have something to say.
 *
 * A notice with no words is one nobody could read.
 */
export const standing = (notices: readonly Notice[]): readonly Notice[] =>
  notices.filter((notice) => notice.says !== '')

/** What a notice counts against, for the ones that count anything. */
export const tallyOf = (notice: Notice): { done: number; total: number } | undefined =>
  notice.total === undefined || notice.done === undefined
    ? undefined
    : { done: notice.done, total: notice.total }

/** What one count was doing when it was last read. */
export interface Movement {
  readonly done: number
  readonly rate: number
  readonly at: number
}

/**
 * How fast each count is moving, from what it was doing when it was last read.
 *
 * A count that has gone is forgotten, and one that has just arrived is read once
 * before it has a rate.
 */
export const measured = (
  was: ReadonlyMap<string, Movement>,
  notices: readonly Notice[],
  at: number,
): ReadonlyMap<string, Movement> => {
  const moving = new Map<string, Movement>()
  for (const notice of standing(notices)) {
    const tally = tallyOf(notice)
    if (tally === undefined) continue
    const before = was.get(notice.id)
    const rate = before ? rateOf(before, tally.done, (at - before.at) / 1000) : 0
    moving.set(notice.id, { done: tally.done, rate, at })
  }
  return moving
}

/**
 * How long work runs before it is worth a card, in milliseconds.
 *
 * Most passes are over before a person could read what they were called. A
 * notice somebody asked for does not wait: they are waiting for it.
 */
export const WAIT = 10_000

/**
 * When each notice standing now was first seen. One that has been here keeps
 * the moment it arrived; one that has gone is forgotten.
 */
export const arrivals = (
  was: ReadonlyMap<string, number>,
  notices: readonly Notice[],
  at: number,
): ReadonlyMap<string, number> =>
  new Map(standing(notices).map((notice) => [notice.id, was.get(notice.id) ?? at]))

/** The notices drawn: the ones that have lasted, less the ones put away. */
export const showing = (
  notices: readonly Notice[],
  arrived: ReadonlyMap<string, number>,
  away: ReadonlySet<string>,
  at: number,
  wait: number = WAIT,
): readonly Notice[] =>
  standing(notices).filter(
    (notice) =>
      !away.has(notice.id) && (notice.asked || at - (arrived.get(notice.id) ?? at) >= wait),
  )

/**
 * What is still worth remembering as put away: the notices that are still
 * there. Work that ends and begins again is news, and says so.
 */
export const remembered = (
  away: ReadonlySet<string>,
  notices: readonly Notice[],
): ReadonlySet<string> => {
  const here = new Set(standing(notices).map((notice) => notice.id))
  return new Set([...away].filter((id) => here.has(id)))
}
