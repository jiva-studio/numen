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
  readonly written: string
  readonly path: string
  readonly title: string
  readonly body: string
  readonly label: string
  readonly hasPoints: boolean
  readonly isAmbiguous: boolean
  readonly error: string
}

/** The notes linked to one deck. */
export interface DeckNeighbourhood {
  readonly notes: readonly Neighbour[]
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
