/**
 * Type declarations for the flashcards deck tab domain.
 */
import type { Surrounds, Value } from '../../../types'
import type { BufferSection } from './bufferSection'

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

/** A deck as the window holds it. */
export interface BufferDeck extends Surrounds {
  readonly cards: readonly BufferCard[]
  readonly sections: readonly BufferSection[]
}

/** A deck of no cards, which is what a file nothing has been written to holds. */
export const NO_DECK: BufferDeck = { preamble: '', cards: [], sections: [], tail: '' }
