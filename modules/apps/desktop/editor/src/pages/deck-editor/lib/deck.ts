/**
 * Flashcard deck domain model, conversions, and operations.
 */
export type { BufferCard, BufferDeck, BufferSection } from '../types'
export { NO_DECK } from '../types'
export {
  deserializeVaultDeck,
  serializeBufferDeckToString,
  deserializeBufferDeckFromString,
  serializeBufferCardsToVaultCards,
  serializeBufferSectionsToVaultSections,
} from './serialize'
export {
  applyHead,
  applyName,
  cardsOf,
  pathOfCut,
  sameDeck,
  sameOffers,
  sectionsOf,
  stencilsOf,
} from './view'
export { addCard, dropCard, fillCard, removeCard } from './cards'
export { addSection, removeSection, renameSection } from './sections'
