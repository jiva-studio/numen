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
  drawn: readonly Notice[],
  room: number = ROOM,
): { shown: readonly Notice[]; over: number } => {
  if (drawn.length <= room) return { shown: drawn, over: 0 }
  const spare = drawn.filter(
    (notice) =>
      notice.stay !== undefined && notice.stay !== 'holds' && notice.tone !== 'alarm',
  )
  const away = new Set(spare.slice(0, drawn.length - room).map((notice) => notice.id))
  return { shown: drawn.filter((notice) => !away.has(notice.id)), over: away.size }
}
