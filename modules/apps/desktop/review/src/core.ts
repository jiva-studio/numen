/**
 * Everything this window asks of the application.
 *
 * Nothing here draws: it is the client, and the words the answers arrive in.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Rating, ReviewService } from '@numen/protocol'

export const review = createClient(
  ReviewService,
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
  readonly heading: string
  readonly front: string
  readonly back: string
  readonly seen: boolean
  readonly due: string
}

/** What a person wrote under one of a card's fields. */
export interface Held {
  readonly field: string
  readonly text: string
}

/** The name of a deck, as it is shown: the file's, without folders or suffix. */
export const deckName = (path: string): string => {
  const last = path.split('/').pop() ?? path
  return last.replace(/\.[^.]+$/, '')
}
