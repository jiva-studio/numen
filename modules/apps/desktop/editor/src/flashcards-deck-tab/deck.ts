/**
 * Flashcard deck domain model, conversions, and operations.
 */
export type { BufferCard, BufferDeck, BufferSection } from './types'
export { NO_DECK } from './types'
export { cardsOf, deckBodyOf, deckIn, deckOf, sectionsOf } from './serialize'
export {
  applyHead,
  applyName,
  drawnOf,
  drawnSectionsOf,
  pathOfCut,
  sameDeck,
  sameOffers,
  stencilsOf,
} from './drawn'
export {
  addCard,
  addSection,
  dropCard,
  fillCard,
  removeCard,
  removeSection,
  renameSection,
} from './mutations'
