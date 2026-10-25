/** How many cards stand at once, and how many fold away behind them. */

import type { Notice } from './notice'

/** How many cards stand at once. */
export const ROOM = 4

/**
 * The cards that stand, and how many are folded away behind them.
 *
 * What folds is what has been said and has gone right. Work, what is so, and
 * anything that stopped badly stand however many of them there are.
 */
export const foldNotices = (
  notices: readonly Notice[],
  room: number = ROOM,
): { visible: readonly Notice[]; overflowCount: number } => {
  if (notices.length <= room) return { visible: notices, overflowCount: 0 }
  const spare = notices.filter(
    (notice) => notice.stay !== undefined && notice.stay !== 'holds' && notice.tone !== 'alarm',
  )
  const away = new Set(spare.slice(0, notices.length - room).map((notice) => notice.id))
  return { visible: notices.filter((notice) => !away.has(notice.id)), overflowCount: away.size }
}
