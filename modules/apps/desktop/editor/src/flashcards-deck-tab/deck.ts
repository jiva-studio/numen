/**
 * Flashcard deck domain model, conversions, and operations.
 */
export type { BufferCard, BufferDeck, BufferSection } from './types'
export { NO_DECK } from './types'
export { cardsOf, deckBodyOf, deckIn, deckOf, sectionsOf } from './serialize'
export {
  drawnOf,
  drawnSectionsOf,
  headed,
  named,
  pathOfCut,
  sameDeck,
  sameOffers,
  stencilsOf,
} from './drawn'
export {
  added,
  dropped,
  filled,
  removed,
  sectionAdded,
  sectionGone,
  sectionNamed,
} from './mutations'
