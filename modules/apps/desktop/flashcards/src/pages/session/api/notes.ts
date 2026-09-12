/**
 * What the window asks about the notes a deck is joined to.
 *
 * Nothing here draws. What arrives is turned into the plain values the panel
 * carries, so what the schema calls things stops at this file.
 */
import { formatErrorCodeMessage } from '@numen/wire'

import { cards } from '@/shared/clients'

/** One note the deck is joined to. */
export interface Neighbour {
  /** The link as it is written, which is all a name answering to nothing has. */
  readonly written: string
  /** Empty where the name answers to no note. */
  readonly path: string
  readonly title: string
  readonly body: string
  /** What the person called the relationship, where they called it anything. */
  readonly label: string
  /** The deck points at it; otherwise it points at the deck. */
  readonly points: boolean
  /** Several notes answer to the name, and this is the nearest. */
  readonly ambiguous: boolean
  /** Why there is no text, empty where the text is here. */
  readonly refusal: string
}

/** What one deck stands among. */
export interface DeckNeighbourhood {
  readonly notes: readonly Neighbour[]
  /** How many at the end came named and not read. */
  readonly unread: number
}

/** The notes one deck is joined to, in the order they are read. */
export const around = async (vault: string, deck: string): Promise<DeckNeighbourhood> => {
  const answer = await cards.getDeckNeighbourhood({ vault, deck })
  return {
    notes: answer.notes.map((one) => ({
      written: one.written,
      path: one.path,
      title: one.title,
      body: one.body,
      label: one.label,
      points: one.points,
      ambiguous: one.ambiguous,
      refusal: formatErrorCodeMessage(one.refusal),
    })),
    unread: answer.unread,
  }
}
