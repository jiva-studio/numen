/** A preset as this screen holds one: how it schedules, and what its day holds. */
import type { Settings as WireSettings } from '../../../lib/wire/settings'
import type { BudgetKeys } from '../vault/types'

/** A budget taking no part in the day, which nothing is weighed against. */
export const CLOSES_NOTHING: BudgetKeys = { new: '', reviews: '', minutes: '' }

/** How the decks pointing at one preset are scheduled, as far as this screen reads it. */
export type Settings = Pick<
  WireSettings,
  | 'goal'
  | 'byDate'
  | 'minutesADay'
  | 'newADay'
  | 'reviewsADay'
  | 'retention'
  | 'load'
  | 'hasEvenLoad'
>

/**
 * What one day holds under a preset, the day of the week having had its say:
 * cards of each kind, and how long the day runs.
 */
export interface Budget {
  new: number
  reviews: number
  minutes: number
}

/** One preset, the decks it schedules, and its contents today. */
export interface Preset {
  readonly path: string
  readonly name: string
  readonly settings: Settings | null
  readonly decks: readonly string[]
  readonly named: number
  readonly reviewCards: number
  readonly cards: number
  readonly budget: Budget
  readonly closes: BudgetKeys
  readonly reviewed: number
  readonly reviewedNew: number
  readonly reviewedReviews: number
  readonly took: number
  readonly paused: string
  readonly wrong: string
}
