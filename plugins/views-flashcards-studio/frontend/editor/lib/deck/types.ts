/**
 * Type declarations for the flashcards deck tab domain.
 */
import type { Surrounds, Value } from '../../../types'

/**
 * One card as the editor holds it: the file's content, addressed by its identity.
 */
export interface BufferCard {
  readonly id: string
  readonly cardId: string
  readonly section: string | null
  readonly heading: string
  readonly cardModelLink: string
  readonly cardModelPath: string
  readonly preamble: string
  readonly values: readonly Value[]
}

/** One section as the window holds it, under an identity of its own. */
export interface BufferSection {
  readonly id: string
  readonly name: string
  readonly preamble: string
}

/** A deck as the window holds it. */
export interface BufferDeck extends Surrounds {
  readonly cards: readonly BufferCard[]
  readonly sections: readonly BufferSection[]
}

/** A deck of no cards, which is what a file nothing has been written to holds. */
export const NO_DECK: BufferDeck = { preamble: '', cards: [], sections: [], tail: '' }
