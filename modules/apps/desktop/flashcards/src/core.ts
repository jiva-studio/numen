/**
 * Everything this window asks of the application.
 *
 * Nothing here draws: it is the client, and the words the answers arrive in.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Rating, FlashcardsService } from '@numen/protocol'

export const cards = createClient(
  FlashcardsService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

/** How well a card came back. A person says which of the four. */
export type Said = 'again' | 'hard' | 'good' | 'easy'

/** The four, in the order they are offered and answered by number. */
export const said: readonly Said[] = ['again', 'hard', 'good', 'easy']

/** What each of them is called on the button that says it. */
export const called: Readonly<Record<Said, string>> = {
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
}

/** What the schema calls each of them. */
export const rated: Readonly<Record<Said, Rating>> = {
  again: Rating.AGAIN,
  hard: Rating.HARD,
  good: Rating.GOOD,
  easy: Rating.EASY,
}

/** One deck's share of what a vault owes. */
export interface DeckOwing {
  readonly deck: string
  readonly faces: number
  readonly due: number
  readonly new: number
}

/** One vault, and what its cards come to today. */
export interface Owing {
  readonly vaultId: string
  readonly name: string
  readonly path: string
  readonly faces: number
  readonly due: number
  readonly new: number
  readonly decks: readonly DeckOwing[]
  /** Why nothing was counted, where nothing was. */
  readonly unread: string
}

/** One card face as it is put to a person. */
export interface Asked {
  readonly deck: string
  readonly section: string
  readonly card: string
  readonly face: string
  readonly front: string
  readonly back: string
  readonly seen: boolean
  /** Where each of the four would leave it. */
  readonly ahead: Ahead | null
}

/**
 * How long each of the four would leave the card, in seconds from when it was
 * asked. A person choosing between the four is choosing between these.
 */
export type Ahead = Readonly<Record<Said, number>>

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
