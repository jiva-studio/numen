/**
 * What a vault owes today: the vault, the decks in it, and the presets
 * scheduling them.
 */
import type { StopReason } from '@numen/protocol'

/** One deck's share of what a vault owes. */
export interface DeckCardsDue {
  readonly deck: string
  readonly faces: number
  readonly due: number
  readonly new: number
  /**
   * How many of its card faces stand learned now, under the rule the preset
   * scheduling it counts by.
   */
  readonly learned: number
  /**
   * How many of its card faces nobody has answered at all. A deck every face of
   * which is one of these has nothing that can come round until something
   * begins them.
   */
  readonly unbegun: number
}

/** One preset of a vault, and what the day comes to under it. */
export interface PresetCardsDue {
  /** The note it stands in, and empty for the decks naming no preset. */
  readonly preset: string
  /** What it is called, and empty where nothing names the note. */
  readonly title: string
  /** How many decks point at it, which is none for a preset nothing points at. */
  readonly decks: number
  /** How many card faces stand in those decks. A deck holding none still points here. */
  readonly cards: number
  /**
   * What the day leaves under it, already held to the budgets that close it.
   * It is what a session over this preset asks, and is printed as it stands.
   */
  readonly owed: number
  /**
   * Cards answered under it since the day opened, and the minutes they took.
   * The two beside the total divide it the way a budget does, so each is
   * weighed against the budget of its own kind.
   */
  readonly answered: number
  readonly answeredNew: number
  readonly answeredReviews: number
  readonly took: number
  /**
   * What the day holds under it, the day of the week having had its say: cards
   * of each kind, and how long the day runs. A budget the goal does not name
   * stands as the person left it and closes nothing.
   */
  readonly new: number
  readonly reviews: number
  readonly minutes: number
  /** Which key each budget closes the day on, and empty where it closes none. */
  readonly closes: BudgetKeys
  /**
   * Why it schedules nothing on this day, as the core says it. A preset no deck
   * points at is answered here and nowhere else.
   */
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

/** One vault, and what its cards come to today. */
export interface VaultCardsDue {
  readonly vault: string
  readonly name: string
  readonly path: string
  /**
   * Whether this vault has been counted. Everything below stands at nothing
   * until it has, and nothing under it is read as a vault owing nothing.
   */
  readonly isCounted: boolean
  readonly faces: number
  readonly due: number
  readonly new: number
  readonly decks: readonly DeckCardsDue[]
  readonly presets: readonly PresetCardsDue[]
  /** Why nothing was counted, where nothing was. */
  readonly unread: string
  /**
   * Whether the vault is being read into the index now. Its counts follow when
   * the reading is done.
   */
  readonly isReading: boolean
}
