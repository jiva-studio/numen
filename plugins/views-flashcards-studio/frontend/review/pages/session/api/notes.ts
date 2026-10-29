/**
 * Queries for the notes a deck is linked to.
 *
 * Nothing here renders. The response is converted into the plain values the
 * panel uses, so schema names do not leave this file.
 */
import { formatErrorCodeMessage } from '../../../../cards/wire/errors'

import { cards } from '../../../shared/clients'

/** One note the deck is joined to. */
export interface Neighbour {
  /** The link as it is written, which is all a name answering to nothing has. */
  readonly written: string
  /** Empty where the name answers to no note. */
  readonly path: string
  readonly title: string
  readonly body: string
  /** The label the user gave the relationship, if any. */
  readonly label: string
  /** True when the deck links to the note; false when the note links to the deck. */
  readonly hasPoints: boolean
  /** Several notes answer to the name, and this is the nearest. */
  readonly isAmbiguous: boolean
  /** Why there is no text, empty where the text is here. */
  readonly error: string
}

/** The notes linked to one deck. */
export interface DeckNeighbourhood {
  readonly notes: readonly Neighbour[]
  /** How many at the end came named and not read. */
  readonly unread: number
}

/** The notes one deck is joined to, in the order they are read. */
export const getDeckNeighbourhood = async (
  vault: string,
  deck: string,
): Promise<DeckNeighbourhood> => {
  const answer = await cards.getDeckNeighbourhood({ vault, deck })
  return {
    notes: answer.notes.map((one) => ({
      written: one.address,
      path: one.path,
      title: one.title,
      body: one.body,
      label: one.label,
      hasPoints: one.isPointedAt,
      isAmbiguous: one.isAmbiguous,
      error: formatErrorCodeMessage(one.error),
    })),
    unread: answer.unread,
  }
}
