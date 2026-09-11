/**
 * Tab-level card and section mutations for flashcard deck tabs.
 */
import type { StencilSummary } from '../shared/flashcards/cards'
import {
  addCard,
  addSection,
  dropCard,
  fillCard,
  pathOfCut,
  removeCard,
  removeSection,
  renameSection,
  type BufferCard,
  type BufferDeck,
  type BufferSection,
} from './deck'

export function deckTabActions(
  id: string,
  deckAt: (id: string) => BufferDeck,
  turns: (id: string, deck: BufferDeck) => void,
  getOffers: () => readonly StencilSummary[],
) {
  return {
    adds: (stencil: string, values: readonly string[], section?: string) =>
      turns(id, addCard(deckAt(id), stencil, pathOfCut(getOffers(), stencil), values, section)),
    addCard: (stencil: string, values: readonly string[], section?: string) =>
      turns(id, addCard(deckAt(id), stencil, pathOfCut(getOffers(), stencil), values, section)),
    removes: (card: BufferCard) => turns(id, removeCard(deckAt(id), card)),
    removeCard: (card: BufferCard) => turns(id, removeCard(deckAt(id), card)),
    moves: (card: BufferCard, at: number) => turns(id, dropCard(deckAt(id), card, at)),
    moveCard: (card: BufferCard, at: number) => turns(id, dropCard(deckAt(id), card, at)),
    writes: (card: BufferCard, field: string, nth: number, text: string) =>
      turns(id, fillCard(deckAt(id), card, field, nth, text)),
    writeCardField: (card: BufferCard, field: string, nth: number, text: string) =>
      turns(id, fillCard(deckAt(id), card, field, nth, text)),
    addsSection: (name: string) => turns(id, addSection(deckAt(id), name)),
    addSection: (name: string) => turns(id, addSection(deckAt(id), name)),
    namesSection: (section: BufferSection, name: string) =>
      turns(id, renameSection(deckAt(id), section, name)),
    renameSection: (section: BufferSection, name: string) =>
      turns(id, renameSection(deckAt(id), section, name)),
    removesSection: (section: BufferSection) => turns(id, removeSection(deckAt(id), section)),
    removeSection: (section: BufferSection) => turns(id, removeSection(deckAt(id), section)),
  }
}
