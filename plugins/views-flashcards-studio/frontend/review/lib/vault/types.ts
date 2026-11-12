/**
 * What a vault has due today: the vault, the decks in it, and the presets that
 * schedule them.
 */
import type { StopReason } from '../../../lib/stop'

/** One deck's share of what a vault has due. */
export interface DeckCardsDue {
  readonly deck: string
  readonly reviewCards: number
  readonly due: number
  readonly new: number
  readonly learned: number
  readonly unbegun: number
}

/** One preset of a vault, and what the day evaluates to under it. */
export interface PresetCardsDue {
  readonly preset: string
  readonly title: string
  readonly decks: number
  readonly cards: number
  readonly pending: number
  readonly reviewed: number
  readonly reviewedNew: number
  readonly reviewedReviews: number
  readonly took: number
  readonly new: number
  readonly reviews: number
  readonly minutes: number
  readonly closes: BudgetKeys
  readonly stopsOn: StopReason
}

/**
 * The key each of the three budgets closes the day on, as the preset writes it.
 * An empty one is a budget taking no part, and nothing is weighed against it.
 */
export interface BudgetKeys {
  readonly new: string
  readonly reviews: string
  readonly minutes: string
}

/** Which vault it is: its id, its name and where it lies. */
interface VaultIdentity {
  readonly vault: string
  readonly name: string
  readonly path: string
}

/** What the cards of a vault come to today. */
interface VaultTotals {
  readonly reviewCards: number
  readonly due: number
  readonly new: number
}

/** One vault, and what its cards come to today. */
export interface VaultCardsDue {
  readonly identity: VaultIdentity
  readonly isCounted: boolean
  readonly totals: VaultTotals
  readonly decks: readonly DeckCardsDue[]
  readonly presets: readonly PresetCardsDue[]
  readonly unread: string
  readonly isIndexing: boolean
}
