/** How long a notice stands, and which of the ones handed in are drawn. */

import { readable, type Notice } from './notice'

/**
 * How long work runs before it is worth a card, in milliseconds.
 *
 * Most passes are over before a person could read what they were called. A
 * notice somebody asked for does not wait: they are waiting for it.
 */
export const WAIT = 10_000

/** How long the shortest thing worth saying stands, in milliseconds. */
export const SETTLE = 4_000

/** How much longer it stands for each word it carries, in milliseconds. */
export const PER_WORD = 400

/** The length past which a notice is not read in passing. */
export const TOO_MUCH = 20

/**
 * How long a notice stands to be read, in milliseconds.
 *
 * Longer words are read for longer. Past twenty of them the time never runs
 * out: a list of names is something a person acts on, and it waits for them.
 */
export const dwellOf = (says: string, about = ''): number => {
  const words = `${says} ${about}`.split(/\s+/).filter((word) => word !== '')
  if (words.length > TOO_MUCH) return Infinity
  return SETTLE + words.length * PER_WORD
}

/**
 * When each notice standing now was first seen. One that has been here keeps
 * the moment it arrived; one that has gone is forgotten.
 */
export const getArrivalTimes = (
  was: ReadonlyMap<string, number>,
  notices: readonly Notice[],
  at: number,
): ReadonlyMap<string, number> =>
  new Map(readable(notices).map((notice) => [notice.id, was.get(notice.id) ?? at]))

/** Whether a notice has stood long enough to have been read. */
const isRead = (notice: Notice, firstSeen: ReadonlyMap<string, number>, at: number): boolean =>
  at - (firstSeen.get(notice.id) ?? at) >= dwellOf(notice.says, notice.about)

/** The notices drawn: the ones that have lasted, less the ones put away. */
export const getShownNotices = (
  notices: readonly Notice[],
  firstSeen: ReadonlyMap<string, number>,
  away: ReadonlySet<string>,
  at: number,
  wait: number = WAIT,
): readonly Notice[] =>
  readable(notices).filter((notice) => {
    if (away.has(notice.id)) return false
    if (notice.stay === 'read') return !isRead(notice, firstSeen, at)
    return notice.isAsked || at - (firstSeen.get(notice.id) ?? at) >= wait
  })

/** The notices that have been read, and whose caller may dismiss them. */
export const getFinishedNotices = (
  notices: readonly Notice[],
  firstSeen: ReadonlyMap<string, number>,
  at: number,
): readonly string[] =>
  readable(notices)
    .filter((notice) => notice.stay === 'read' && isRead(notice, firstSeen, at))
    .map((notice) => notice.id)
