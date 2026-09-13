/**
 * Tab-level card and section mutations for flashcard deck tabs.
 */
import type { StencilSummary, Value } from '@/entities/deck'
import {
  addCard,
  addSection,
  dropCard,
  fillCard,
  pathOfCut,
  removeCard,
  removeSection,
  renameSection,
  type BufferDeck,
} from '../lib/deck'

export function createDeckTabActions(
  id: string,
  deckAt: (id: string) => BufferDeck,
  updateDeck: (id: string, deck: BufferDeck) => void,
  getOffers: () => readonly StencilSummary[],
) {
  return {
    addCard: (stencil: string, values: readonly Value[], section: string | null = null) =>
      updateDeck(
        id,
        addCard(deckAt(id), stencil, pathOfCut(getOffers(), stencil), values, section),
      ),
    removeCard: (card: string) => updateDeck(id, removeCard(deckAt(id), card)),
    moveCard: (card: string, at: string | null) => updateDeck(id, dropCard(deckAt(id), card, at)),
    writeCardField: (card: string, field: string, nth: number, text: string) =>
      updateDeck(id, fillCard(deckAt(id), card, field, nth, text)),
    addSection: (name: string) => updateDeck(id, addSection(deckAt(id), name)),
    renameSection: (section: string, name: string) =>
      updateDeck(id, renameSection(deckAt(id), section, name)),
    removeSection: (section: string) => updateDeck(id, removeSection(deckAt(id), section)),
  }
}
