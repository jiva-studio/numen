/** How far a preset is through its day, and what its decks still offer. */
import type { DeckCardsDue } from '../../../entities/vault'

import type { Preset } from '../types'

/**
 * Whether starting a session on one deck is allowed: it has cards due today, and the
 * preset scheduling it schedules something.
 */
export const canStart = (deck: DeckCardsDue, presets: ReadonlyMap<string, Preset>): boolean =>
  deck.due + deck.new > 0 && !presets.get(deck.deck)?.paused

/**
 * Whether a deck holds nothing that can ever be presented: every review card in
 * it is unbegun, and the preset scheduling it begins none a day.
 *
 * It is a fact about the material, and not a reason the preset is stopped. The
 * preset schedules; there is nothing here for it to schedule.
 */
export const hasNothingToBegin = (deck: DeckCardsDue, preset: Preset | undefined): boolean =>
  deck.reviewCards > 0 && deck.unbegun === deck.reviewCards && preset?.budget.new === 0

/**
 * How far through its day a preset is: the answered cards against the
 * cards the day holds, and the time taken against the minutes the day runs,
 * whichever of the two is further along.
 *
 * A day answered beyond its budget yields a value above one.
 */
export const getSpentShare = (one: Preset): number => {
  // Only a budget that closes the day is weighed, and each is weighed
  // against the answers of its own kind. A preset steered by its minutes
  // leaves its card counts unweighed.
  const shares = [
    one.closes.new ? share(one.reviewedNew, one.budget.new) : 0,
    one.closes.reviews ? share(one.reviewedReviews, one.budget.reviews) : 0,
    one.closes.minutes ? share(one.took, one.budget.minutes) : 0,
  ]
  return Math.max(0, ...shares)
}

/** What a count evaluates to against a budget. A budget of nothing is no share. */
const share = (count: number, budget: number): number => (budget > 0 ? count / budget : 0)

/**
 * Whether a preset's day is spent, which is what leaves every deck under it
 * nothing more to ask however much those decks still hold.
 */
export const isSpent = (one: Preset): boolean => getSpentShare(one) >= 1

/**
 * How much of a deck is learned, as a share of its review cards, and null for
 * a deck holding none: a share of nothing is not defined.
 */
export const getLearnedShare = (deck: DeckCardsDue): number | null =>
  deck.reviewCards > 0 ? deck.learned / deck.reviewCards : null
