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
  const markCounts = new Map<string, number>()
  for (const card of readItem.cards) markCounts.set(card.mark, (markCounts.get(card.mark) ?? 0) + 1)
  const sections = readItem.sections.map((section) => ({
    id: generateBufferId(),
    name: section.name,
    preamble: section.preamble,
  }))
  return {
    preamble: readItem.preamble,
    cards: readItem.cards.map((card) => ({
      id: card.mark && markCounts.get(card.mark) === 1 ? card.mark : generateBufferId(),
      mark: card.mark,
      section: sections[card.sectionIndex ?? -1]?.id ?? null,
      heading: card.heading,
      stencilLink: card.stencilLink,
      stencilPath: card.stencilPath,
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
      mark: card.mark,
      section: card.section,
      heading: card.heading,
      stencilLink: card.stencilLink,
      stencilPath: card.stencilPath,
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
    ({ mark, section, heading, stencilLink, stencilPath, preamble, values }) => ({
      mark,
      sectionIndex: sectionIndices.get(section) ?? null,
      heading,
      stencilLink,
      stencilPath,
      preamble,
      values,
    }),
  )
}

/** The sections of a deck, in the shape the vault takes them. */
export const serializeBufferSectionsToVaultSections = (deck: BufferDeck): readonly VaultSection[] =>
  deck.sections.map(({ name, preamble }) => ({ name, preamble }))
