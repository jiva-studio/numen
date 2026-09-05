/**
 * Everything this window asks of the application.
 *
 * Nothing here draws: it is the client, and the words the answers arrive in.
 */
import { createClient } from '@connectrpc/connect'
import { Goal as Goals, Rating, FlashcardsService, WindowService } from '@numen/protocol'
import type { StopReason } from '@numen/protocol'
import { namesOf } from '@numen/wire'
import { transport } from './transport'

/**
 * What this window asks, presets among it. Every question names the vault it is
 * about: this window is over all of them at once.
 */
export const cards = createClient(FlashcardsService, transport)

/** This window itself, which is the one cards are run in and not the editor. */
export const itself = createClient(WindowService, transport)

/** The window every question about a window names. */
export const WINDOW = 'review'

/** How well a card came back. A person says which of the four. */
export type Grade = 'again' | 'hard' | 'good' | 'easy'

/** The four, in the order they are offered and answered by number. */
export const grades: readonly Grade[] = ['again', 'hard', 'good', 'easy']

/** What each of them is called on the button that says it. */
export const called: Readonly<Record<Grade, string>> = {
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
}

/**
 * What each of them is called in this window's own words. Keyed by the schema,
 * so a rating added to it has to be given a word here before this compiles.
 */
const graded: Readonly<Record<Rating, Grade | null>> = {
  [Rating.UNSPECIFIED]: null,
  [Rating.AGAIN]: 'again',
  [Rating.HARD]: 'hard',
  [Rating.GOOD]: 'good',
  [Rating.EASY]: 'easy',
}

/** What the schema calls each of them, read off the words above. */
export const rated: Readonly<Record<Grade, Rating>> = namesOf<Grade, Rating>(graded)

/** Which value the one control of a preset steers. */
export type Goal = 'minutes' | 'retention' | 'date'

/** The goal in this window's own words. A preset naming none aims at minutes. */
export const goalOf: Readonly<Record<Goals, Goal>> = {
  [Goals.UNSPECIFIED]: 'minutes',
  [Goals.MINUTES_A_DAY]: 'minutes',
  [Goals.RETENTION]: 'retention',
  [Goals.BY_DATE]: 'date',
}

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
   * It is what a sitting over this preset asks, and is printed as it stands.
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
  readonly counted: boolean
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
  readonly reading: boolean
}

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
export const ahead = (seconds: number): string => {
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

/** The name of a deck, as it is shown: the file's, without folders or suffix. */
export const deckName = (path: string): string => {
  const last = path.split('/').pop() ?? path
  return last.replace(/\.[^.]+$/, '')
}
