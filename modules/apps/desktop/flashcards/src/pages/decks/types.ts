/** A preset as this screen holds one: how it schedules, and what its day holds. */
import type { Goal as Goals } from '@numen/protocol'
import type { Goal } from '@numen/wire'
import type { BudgetKeys } from '@/entities/vault'

/** A budget taking no part in the day, which nothing is weighed against. */
export const CLOSES_NOTHING: BudgetKeys = { new: '', reviews: '', minutes: '' }

/** How the decks pointing at one preset are scheduled. */
export interface Settings {
  goal: Goal
  byDate: string
  minutesADay: number
  newADay: number
  reviewsADay: number
  retention: number
  /** What each day of the week carries, in per cent, under the day's own name. */
  load: Record<string, number>
  hasEvenLoad: boolean
}

/** The same, as the schema carries them. */
export interface SettingsMessage extends Omit<Settings, 'goal'> {
  goal: Goals
}

/**
 * What one day holds under a preset, the day of the week having had its say:
 * cards of each kind, and how long the day runs.
 */
export interface Budget {
  new: number
  reviews: number
  minutes: number
}

/** One preset, the decks it schedules, and what it holds today. */
export interface Preset {
  /** The note the settings were read from, and empty for the defaults. */
  readonly path: string
  readonly name: string
  /** How its decks are scheduled, and null where no deck points at it. */
  readonly settings: Settings | null
  /** The decks it schedules, by the path each is filed under. */
  readonly decks: readonly string[]
  /** How many decks name it, whatever they hold. */
  readonly named: number
  /** How many card faces stand in those decks. */
  readonly faces: number
  /**
   * What starting a session on it would ask, as the count gives it: the cards its
   * decks owe today, already held to the budgets that close the day.
   */
  readonly cards: number
  /** What the day holds under it, which is what today is weighed against. */
  readonly budget: Budget
  /** Which key each of those budgets closes on, and empty where it closes none. */
  readonly closes: BudgetKeys
  /**
   * Cards answered under it since the day opened, and the minutes they took.
   * The two beside the total divide it the way a budget does, so each is
   * weighed against the budget of its own kind.
   */
  readonly answered: number
  readonly answeredNew: number
  readonly answeredReviews: number
  readonly took: number
  /** Why it schedules nothing, and empty while it schedules something. */
  readonly paused: string
  /**
   * What is wrong with the preset, in the words to show, and empty where
   * nothing is: what the file said that could not be read, or why the file
   * itself could not be.
   */
  readonly wrong: string
}
