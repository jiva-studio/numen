/** One review card as presented for review, and where each of the four ratings would schedule it. */
import type { Grade } from './grades'

/** One review card as presented for review. */
export interface ReviewCard {
  readonly deck: string
  readonly section: string
  /** What the card is known by. */
  readonly cardId: string
  readonly template: string
  /** What the card's heading shows, which is the first line of its first field. */
  readonly heading: string
  readonly front: string
  readonly back: string
  readonly isNew: boolean
  /** Where each of the four would leave it. */
  readonly ahead: Intervals | null
}

/**
 * How long each of the four ratings would schedule the card out, in seconds from
 * when it was requested. The choice between the four ratings is a choice between
 * these.
 */
export type Intervals = Readonly<Record<Grade, number>>

/**
 * A length of time at the coarsest unit that reads naturally: minutes inside an
 * hour, hours inside a day, then days, months and years.
 *
 * A card due in eleven minutes and one due in twelve are the same answer for the
 * choice between ratings, and the shorter the text the faster the four are read.
 */
export const getTimeAhead = (seconds: number): string => {
  const minutes = Math.max(1, Math.round(seconds / 60))
  if (minutes < 60) return `${minutes}m`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.round(hours / 24)
  if (days < 30) return `${days}d`
  const months = Math.round(days / 30)
  if (months < 12) return `${months}mo`
  return `${Math.round(days / 365)}y`
}
