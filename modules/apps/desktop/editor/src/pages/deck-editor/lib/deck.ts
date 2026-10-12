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
  applyReadHeadings,
  applyScreenIds,
  createDeckCards,
  findStencilPath,
  areDecksEqual,
  areOffersEqual,
  createDeckSections,
  createStencilChoices,
} from './view'
export { addCard, dropCard, replaceCardValue, removeCard } from './cards'
export { addSection, removeSection, renameSection } from './sections'
