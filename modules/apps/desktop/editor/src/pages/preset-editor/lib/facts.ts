/**
 * Which facts the preset tab states about a place on the curve and about the
 * material, and in what order. How a fact is worded is `words.ts`.
 */
import type { Goal } from '../types'

/** What a place on the curve buys. */
export interface BuyInput {
  value: number
  reviews: number
  minutes: number
  horizon: number
  /** Null is nothing overdue, and -1 a pile the days projected do not clear. */
  clears: number | null
  /** How many card faces no pace reaches by the day, out of how many there are. */
  short: number
  cards: number
}

export type BuyFact =
  | { kind: 'daysOff'; days: number }
  | { kind: 'minutesADay'; minutes: number }
  | { kind: 'remembered'; share: number }
  | { kind: 'cardsASession'; cards: number }
  | { kind: 'cannotGetThere'; short: number; cards: number }
  | { kind: 'overdueGone'; days: number }
  | { kind: 'overdueStays'; horizon: number }

/**
 * The facts of what a value buys under a goal. A goal of a date says nothing
 * about the overdue, since every card is to be got through by that day; where
 * nothing is overdue there is no such fact either.
 */
export function getBuyFacts(goal: Goal, at: BuyInput): readonly BuyFact[] {
  if (goal === 'date') {
    const facts: BuyFact[] = [
      { kind: 'daysOff', days: at.value },
      { kind: 'minutesADay', minutes: at.minutes },
    ]
    if (at.short > 0) facts.push({ kind: 'cannotGetThere', short: at.short, cards: at.cards })
    return facts
  }
  const facts: BuyFact[] =
    goal === 'minutes'
      ? [
          { kind: 'minutesADay', minutes: at.value },
          { kind: 'cardsASession', cards: at.reviews },
        ]
      : [
          { kind: 'remembered', share: at.value },
          { kind: 'minutesADay', minutes: at.minutes },
        ]
  if (at.clears === null) return facts
  facts.push(
    at.clears < 0
      ? { kind: 'overdueStays', horizon: at.horizon }
      : { kind: 'overdueGone', days: at.clears },
  )
  return facts
}

export interface MaterialCounts {
  readonly decks: number
  readonly cards: number
  readonly overdue: number
  readonly unbegun: number
}

export type MaterialFact =
  | { kind: 'decks'; count: number }
  | { kind: 'cards'; count: number }
  | { kind: 'overdue'; count: number }
  | { kind: 'unbegun'; count: number }

/** The facts of the material: the decks and cards always, the overdue and new where there are any. */
export function getMaterialFacts(counts: MaterialCounts): readonly MaterialFact[] {
  const facts: MaterialFact[] = [
    { kind: 'decks', count: counts.decks },
    { kind: 'cards', count: counts.cards },
  ]
  if (counts.overdue > 0) facts.push({ kind: 'overdue', count: counts.overdue })
  if (counts.unbegun > 0) facts.push({ kind: 'unbegun', count: counts.unbegun })
  return facts
}

export type LearningFact =
  | { kind: 'notYet' }
  | { kind: 'allToday' }
  | { kind: 'daysToLearn'; days: number }
  | { kind: 'learnedToday'; learned: number; cards: number }

/**
 * When the material is learned, then how much of it stands learned now. With
 * no day to speak of, only what stands learned today is stated; a negative day
 * is a pace whose days run out with a card still to learn.
 */
export function getLearningFacts(
  learns: number | undefined,
  learned: number,
  cards: number,
): readonly LearningFact[] {
  const now: LearningFact = { kind: 'learnedToday', learned, cards }
  if (learns === undefined) return [now]
  if (learns < 0) return [{ kind: 'notYet' }, now]
  if (learns === 0) return [{ kind: 'allToday' }, now]
  return [{ kind: 'daysToLearn', days: learns }, now]
}
