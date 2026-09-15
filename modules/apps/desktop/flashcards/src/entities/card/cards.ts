/** One card face as it is put to a person, and where the four would leave it. */
import type { Grade } from './grades'

/** One card face as it is put to a person. */
export interface CardFace {
  readonly deck: string
  readonly section: string
  /** What the card is known by. */
  readonly mark: string
  readonly face: string
  /** What the card's heading shows, which is the first line of its first field. */
  readonly heading: string
  readonly front: string
  readonly back: string
  readonly seen: boolean
  /** Where each of the four would leave it. */
  readonly ahead: Intervals | null
}

/**
 * How long each of the four would leave the card, in seconds from when it was
 * asked. A person choosing between the four is choosing between these.
 */
export type Intervals = Readonly<Record<Grade, number>>

/**
 * A length of time, at the coarsest a person reads it by: minutes inside an
 * hour, hours inside a day, then days, months and years.
 *
 * A card coming round in eleven minutes and one coming round in twelve are the
 * same answer to the person choosing, and the shorter the word the faster the
 * four are read.
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
