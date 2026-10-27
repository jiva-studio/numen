/**
 * Tab-level card and section mutations for flashcard deck tabs.
 */
import type { CardModelSummary, Value } from '../../cards'
import {
  addCard,
  addSection,
  dropCard,
  replaceCardValue,
  findCardModelPath,
  removeCard,
  removeSection,
  renameSection,
  type BufferDeck,
} from '../lib/deck'

export interface DeckTabActionDeps {
  getDeck: (id: string) => BufferDeck
  updateDeck: (id: string, deck: BufferDeck) => void
  getOffers: () => readonly CardModelSummary[]
}

export function createDeckTabActions(id: string, deps: DeckTabActionDeps) {
  const { getDeck, updateDeck, getOffers } = deps
  return {
    addCard: (cardModel: string, values: readonly Value[], section: string | null = null) =>
      updateDeck(
        id,
        addCard(getDeck(id), cardModel, {
          cardModelPath: findCardModelPath(getOffers(), cardModel),
          values,
          section,
        }),
      ),
    removeCard: (card: string) => updateDeck(id, removeCard(getDeck(id), card)),
    moveCard: (card: string, target: string | null) =>
      updateDeck(id, dropCard(getDeck(id), card, target)),
    writeCardField: (...[card, field, index, text]: [string, string, number, string]) =>
      updateDeck(id, replaceCardValue(getDeck(id), card, { field, index, text })),
    addSection: (name: string) => updateDeck(id, addSection(getDeck(id), name)),
    renameSection: (section: string, name: string) =>
      updateDeck(id, renameSection(getDeck(id), section, name)),
    removeSection: (section: string) => updateDeck(id, removeSection(getDeck(id), section)),
  }
}
