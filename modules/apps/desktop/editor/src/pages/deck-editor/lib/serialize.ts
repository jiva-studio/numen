/**
 * Serialization and conversion between vault deck data and buffer deck models.
 */
import type { VaultCard, VaultDeck, VaultSection } from '@/entities/deck'
import { generateId, type IdMaker } from '@/entities/deck'
import { NO_DECK, type BufferDeck } from '../types'

/**
 * A deck as the vault read it.
 */
export const deserializeVaultDeck = (read: VaultDeck, generateBufferId: IdMaker = generateId): BufferDeck => {
  const held = new Map<string, number>()
  for (const card of read.cards) held.set(card.mark, (held.get(card.mark) ?? 0) + 1)
  const sections = read.sections.map((section) => ({
    id: generateBufferId(),
    name: section.name,
    preamble: section.preamble,
  }))
  return {
    preamble: read.preamble,
    cards: read.cards.map((card) => ({
      id: card.mark && held.get(card.mark) === 1 ? card.mark : generateBufferId(),
      mark: card.mark,
      section: card.sectionIndex === null ? null : (sections[card.sectionIndex]?.id ?? null),
      heading: card.heading,
      stencilLink: card.stencilLink,
      stencilPath: card.stencilPath,
      preamble: card.preamble,
      values: card.values,
    })),
    sections,
    tail: read.tail,
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
  const at = new Map(deck.sections.map((section, index) => [section.id, index]))
  return deck.cards.map(({ mark, section, heading, stencilLink, stencilPath, preamble, values }) => ({
    mark,
    sectionIndex: section === null ? null : (at.get(section) ?? null),
    heading,
    stencilLink,
    stencilPath,
    preamble,
    values,
  }))
}

/** The sections of a deck, in the shape the vault takes them. */
export const serializeBufferSectionsToVaultSections = (deck: BufferDeck): readonly VaultSection[] =>
  deck.sections.map(({ name, preamble }) => ({ name, preamble }))
