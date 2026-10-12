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
  applyHeadings,
  applyIds,
  getCards,
  getCutPath,
  areDecksEqual,
  areOffersEqual,
  getSections,
  getStencils,
} from './view'
export { addCard, dropCard, setCardValue, removeCard } from './cards'
export { addSection, removeSection, renameSection } from './sections'
