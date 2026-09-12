/** How far a preset stands through its day, and what its decks still offer. */
import type { DeckCardsDue } from '@/entities/vault'

import type { Preset } from '../types'

/**
 * Whether starting a session on one deck is offered: it owes something today, and the
 * preset scheduling it schedules something.
 */
export const canStart = (deck: DeckCardsDue, by: ReadonlyMap<string, Preset>): boolean =>
  deck.due + deck.new > 0 && !by.get(deck.deck)?.paused

/**
 * Whether a deck holds nothing that can ever be asked for: every card face in
 * it is one nobody has begun, and the preset scheduling it begins none a day.
 *
 * It is a fact about the material, and not a reason the preset is stopped. The
 * preset schedules; there is nothing here for it to schedule.
 */
export const hasNothingToBegin = (deck: DeckCardsDue, by: Preset | undefined): boolean =>
  deck.faces > 0 && deck.unbegun === deck.faces && by?.budget.new === 0

/**
 * How far through its day a preset stands: what has been answered against the
 * cards the day holds, and what it has taken against the minutes the day runs,
 * whichever of the two is further along.
 *
 * A day answered past what its budget holds stands above one.
 */
export const through = (one: Preset): number => {
  // Only a budget that closes the day is weighed against, and each is weighed
  // against what was answered of its own kind. A preset steered by its minutes
  // keeps its card counts as the person left them.
  const shares = [
    one.closes.new ? share(one.answeredNew, one.budget.new) : 0,
    one.closes.reviews ? share(one.answeredReviews, one.budget.reviews) : 0,
    one.closes.minutes ? share(one.took, one.budget.minutes) : 0,
  ]
  return Math.max(0, ...shares)
}

/** What a count comes to against a budget. A budget of nothing is no share. */
const share = (count: number, budget: number): number => (budget > 0 ? count / budget : 0)

/**
 * Whether a preset's day is spent, which is what leaves every deck under it
 * nothing more to ask however much those decks still hold.
 */
export const isSpent = (one: Preset): boolean => through(one) >= 1

/**
 * How much of a deck stands learned, as a share of its card faces, and null for
 * a deck holding none: a share of nothing is no share.
 */
export const getLearnedShare = (deck: DeckCardsDue): number | null =>
  deck.faces > 0 ? deck.learned / deck.faces : null
