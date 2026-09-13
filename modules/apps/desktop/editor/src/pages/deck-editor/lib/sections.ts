/**
 * Pure mutation operations for deck sections.
 */
import { generateId, type IdMaker } from '@/entities/deck'
import type { BufferDeck } from '../types'

/** A section made at the end of the deck, holding no card. */
export const addSection = (
  deck: BufferDeck,
  name: string,
  generateSectionId: IdMaker = generateId,
): BufferDeck => ({
  ...deck,
  sections: [...deck.sections, { id: generateSectionId(), name, preamble: '' }],
})

/** A section under another name. */
export const renameSection = (deck: BufferDeck, id: string, name: string): BufferDeck => ({
  ...deck,
  sections: deck.sections.map((section) =>
    section.id === id ? { ...section, name } : section,
  ),
})

/** One piece of a deck's prose after another, with a line between the two. */
const joinProse = (above: string, below: string): string =>
  above && below ? `${above.trimEnd()}\n\n${below}` : above || below

/**
 * A section taken out of the deck.
 */
export const removeSection = (deck: BufferDeck, id: string): BufferDeck => {
  const at = deck.sections.findIndex((section) => section.id === id)
  if (at === -1) return deck
  const going = deck.sections[at]
  const above = deck.sections[at - 1]
  return {
    ...deck,
    preamble: above ? deck.preamble : joinProse(deck.preamble, going?.preamble ?? ''),
    sections: deck.sections.flatMap((section) => {
      if (section.id === id) return []
      if (above && section.id === above.id) {
        return [{ ...section, preamble: joinProse(section.preamble, going?.preamble ?? '') }]
      }
      return [section]
    }),
    cards: deck.cards.map((card) =>
      card.section === id ? { ...card, section: above?.id ?? null } : card,
    ),
  }
}
