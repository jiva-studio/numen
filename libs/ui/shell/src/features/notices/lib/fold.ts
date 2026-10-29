/** How many cards stand at once, and how many fold away behind them. */

import type { Notice } from './notice'

/** How many cards stand at once. */
export const ROOM = 4

/**
 * The cards that remain visible, and how many are folded away behind them.
 *
 * Cards for completed work fold. Work in progress, current status, and anything
 * that ended badly remain visible regardless of count.
 */
export const foldNotices = (
  notices: readonly Notice[],
  room: number = ROOM,
): { visible: readonly Notice[]; overflowCount: number } => {
  if (notices.length <= room) return { visible: notices, overflowCount: 0 }
  const spare = notices.filter(
    (notice) => notice.stay !== undefined && notice.stay !== 'holds' && notice.severity !== 'error',
  )
  const away = new Set(spare.slice(0, notices.length - room).map((notice) => notice.id))
  return { visible: notices.filter((notice) => !away.has(notice.id)), overflowCount: away.size }
}
