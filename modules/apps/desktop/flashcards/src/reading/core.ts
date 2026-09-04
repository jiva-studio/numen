/**
 * What the window asks about the notes a deck is joined to.
 *
 * Nothing here draws. What arrives is turned into the plain values the panel
 * carries, so what the schema calls things stops at this file.
 */
import { Refusal } from '@numen/protocol'

import { cards } from '../core'
import { REFUSED } from './words'
import type { Refused } from './words'

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
export interface Around {
  readonly notes: readonly Neighbour[]
  /** How many at the end came named and not read. */
  readonly unread: number
}

/** The notes one deck is joined to, in the order they are read. */
export const around = async (vault: string, deck: string): Promise<Around> => {
  const answer = await cards.around({ vault, deck })
  return {
    notes: answer.notes.map((one) => ({
      written: one.written,
      path: one.path,
      title: one.title,
      body: one.body,
      label: one.label,
      points: one.points,
      ambiguous: one.ambiguous,
      refusal: said(one.refusal),
    })),
    unread: answer.unread,
  }
}

/** Why a note has no text, in words a person reads, and nothing where it has. */
export const said = (refusal: Refusal | undefined): string =>
  refusal === undefined ? '' : REFUSED[refused[refusal]]

/** What the schema calls each refusal this panel can be given. */
const refused: Record<Refusal, Refused> = {
  [Refusal.UNSPECIFIED]: 'unreadable',
  [Refusal.MISSING]: 'missing',
  [Refusal.NOT_A_NOTE]: 'notANote',
  [Refusal.NOT_TEXT]: 'notText',
  [Refusal.TOO_LARGE]: 'tooLarge',
  [Refusal.BODY_REFUSED]: 'unreadable',
  [Refusal.UNREADABLE]: 'unreadable',
  [Refusal.OCCUPIED]: 'unreadable',
  [Refusal.UNNAMEABLE]: 'unreadable',
  [Refusal.NOT_A_STENCIL]: 'notANote',
  [Refusal.NOT_A_DECK]: 'notANote',
  [Refusal.NOT_A_PRESET]: 'notAPreset',
  [Refusal.DECK_TOO_LARGE]: 'tooLarge',
}
