/**
 * Serialization and conversion between vault deck data and buffer deck models.
 */
import {
  type VaultCard,
  type VaultDeck,
  type VaultSection,
  generateId,
  type IdMaker,
} from '../../cards'
import { NO_DECK, type BufferDeck } from '../types'

/**
 * A deck as the vault read it.
 */
export const deserializeVaultDeck = (
  readItem: VaultDeck,
  generateBufferId: IdMaker = generateId,
): BufferDeck => {
  const cardIdCounts = new Map<string, number>()
  for (const card of readItem.cards)
    cardIdCounts.set(card.cardId, (cardIdCounts.get(card.cardId) ?? 0) + 1)
  const sections = readItem.sections.map((section) => ({
    id: generateBufferId(),
    name: section.name,
    preamble: section.preamble,
  }))
  return {
    preamble: readItem.preamble,
    cards: readItem.cards.map((card) => ({
      id: card.cardId && cardIdCounts.get(card.cardId) === 1 ? card.cardId : generateBufferId(),
      cardId: card.cardId,
      section: sections[card.sectionIndex ?? -1]?.id ?? null,
      heading: card.heading,
      cardModelLink: card.cardModelLink,
      cardModelPath: card.cardModelPath,
      preamble: card.preamble,
      values: card.values,
    })),
    sections,
    tail: readItem.tail,
  }
}

/**
 * A deck as one string, which is what the tab holding it is dirty against.
 */
export const serializeBufferDeckToString = (deck: BufferDeck): string =>
  JSON.stringify({
    preamble: deck.preamble,
    tail: deck.tail,
    sections: deck.sections.map((section) => ({
      id: section.id,
      name: section.name,
      preamble: section.preamble,
    })),
    cards: deck.cards.map((card) => ({
      id: card.id,
      cardId: card.cardId,
      section: card.section,
      heading: card.heading,
      cardModelLink: card.cardModelLink,
      cardModelPath: card.cardModelPath,
      preamble: card.preamble,
      values: card.values.map((value) => ({ field: value.field, text: value.text })),
    })),
  })

/** The deck a string stands for. A string holding nothing is a deck of no cards. */
export const deserializeBufferDeckFromString = (body: string): BufferDeck =>
  body ? (JSON.parse(body) as BufferDeck) : NO_DECK

/**
 * The cards of a deck, in the shape the vault takes them.
 */
export const serializeBufferCardsToVaultCards = (deck: BufferDeck): readonly VaultCard[] => {
  const sectionIndices = new Map<string | null, number>(
    deck.sections.map((section, index) => [section.id, index]),
  )
  return deck.cards.map(
    ({ cardId, section, heading, cardModelLink, cardModelPath, preamble, values }) => ({
      cardId,
      sectionIndex: sectionIndices.get(section) ?? null,
      heading,
      cardModelLink,
      cardModelPath,
      preamble,
      values,
    }),
  )
}

/** The sections of a deck, in the shape the vault takes them. */
export const serializeBufferSectionsToVaultSections = (deck: BufferDeck): readonly VaultSection[] =>
  deck.sections.map(({ name, preamble }) => ({ name, preamble }))
