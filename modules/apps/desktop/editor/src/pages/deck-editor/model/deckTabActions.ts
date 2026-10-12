/**
 * Tab-level card and section mutations for flashcard deck tabs.
 */
import type { StencilSummary, Value } from '@/entities/deck'
import {
  addCard,
  addSection,
  dropCard,
  replaceCardValue,
  findStencilPath,
  removeCard,
  removeSection,
  renameSection,
  type BufferDeck,
} from '../lib/deck'

export function createDeckTabActions(
  id: string,
  getDeck: (id: string) => BufferDeck,
  updateDeck: (id: string, deck: BufferDeck) => void,
  getOffers: () => readonly StencilSummary[],
) {
  return {
    addCard: (stencil: string, values: readonly Value[], section: string | null = null) =>
      updateDeck(
        id,
        addCard(getDeck(id), stencil, findStencilPath(getOffers(), stencil), values, section),
      ),
    removeCard: (card: string) => updateDeck(id, removeCard(getDeck(id), card)),
    moveCard: (card: string, at: string | null) => updateDeck(id, dropCard(getDeck(id), card, at)),
    writeCardField: (card: string, field: string, nth: number, text: string) =>
      updateDeck(id, replaceCardValue(getDeck(id), card, field, nth, text)),
    addSection: (name: string) => updateDeck(id, addSection(getDeck(id), name)),
    renameSection: (section: string, name: string) =>
      updateDeck(id, renameSection(getDeck(id), section, name)),
    removeSection: (section: string) => updateDeck(id, removeSection(getDeck(id), section)),
  }
}
