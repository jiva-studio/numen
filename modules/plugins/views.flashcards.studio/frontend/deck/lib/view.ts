/**
 * UI representation mappings and comparisons for decks and stencils.
 */
import type { DeckCard, DeckSection, Stencil } from '@numen/ui'
import type { StencilSummary, VaultCard } from '../../cards'
import type { BufferDeck } from '../types'
import {
  serializeBufferCardsToVaultCards,
  serializeBufferSectionsToVaultSections,
} from './serialize'

export const createDeckSections = (deck: BufferDeck): readonly DeckSection[] =>
  deck.sections.map(({ id, name }) => ({ id, name }))

/**
 * The cards as the grid draws them, each under the stencil its wikilink resolves to.
 */
export const createDeckCards = (
  deck: BufferDeck,
  offers: readonly StencilSummary[],
): readonly DeckCard[] => {
  const titles = new Map(offers.map((offer) => [offer.path, offer.title]))
  return deck.cards.map((card) => ({
    id: card.id,
    section: card.section,
    stencil: (titles.get(card.stencilPath) ?? card.stencilLink) || null,
    filled: card.values.map((value) => ({ field: value.field, text: value.text })),
  }))
}

/**
 * The stencils a card may be cut by, under the word a card names one by.
 */
export const createStencilChoices = (offers: readonly StencilSummary[]): readonly Stencil[] => {
  const seenTitles = new Set<string>()
  const stencils: Stencil[] = []
  for (const offer of offers) {
    if (!offer.title || seenTitles.has(offer.title)) continue
    seenTitles.add(offer.title)
    stencils.push({ name: offer.title, fields: offer.fields })
  }
  return stencils
}

/** The cards as somebody wrote them, without the heading a write reads back. */
const getWrittenCards = (deck: BufferDeck): readonly Omit<VaultCard, 'heading'>[] =>
  serializeBufferCardsToVaultCards(deck).map(({ heading: _heading, ...card }) => card)

/**
 * Whether two decks read the same.
 */
export const areDecksEqual = (one: BufferDeck, other: BufferDeck): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(serializeBufferSectionsToVaultSections(one)) ===
    JSON.stringify(serializeBufferSectionsToVaultSections(other)) &&
  JSON.stringify(getWrittenCards(one)) === JSON.stringify(getWrittenCards(other))

/**
 * The deck on screen under the headings the file now carries.
 */
export const applyReadHeadings = (screen: BufferDeck, readItem: BufferDeck): BufferDeck => {
  const getHeading = (pos: number): string => readItem.cards[pos]?.heading ?? ''
  if (screen.cards.every((card, index) => card.heading === getHeading(index))) return screen
  return {
    ...screen,
    cards: screen.cards.map((card, index) => ({ ...card, heading: getHeading(index) })),
  }
}

/**
 * A reading of a deck under the identities the window already drew it by.
 */
export const applyScreenIds = (screen: BufferDeck, readItem: BufferDeck): BufferDeck => {
  // Stryker disable next-line ConditionalExpression: findIndex for null id yields -1 either way.
  const seat = (deck: BufferDeck, section: string | null): number =>
    section === null ? -1 : deck.sections.findIndex((each) => each.id === section)

  const sections =
    screen.sections.length === readItem.sections.length
      ? readItem.sections.map((section, at) => {
          const was = screen.sections[at]
          return was && was.name === section.name && was.preamble === section.preamble
            ? { ...section, id: was.id }
            : section
        })
      : readItem.sections

  // Stryker disable next-line ConditionalExpression: index -1 from seat yields null fallback either way.
  const getSectionId = (section: string | null): string | null =>
    section === null ? null : (sections[seat(readItem, section)]?.id ?? null)

  const alongside = screen.cards.length === readItem.cards.length

  const cards = readItem.cards.map((card, at) => {
    const withSection = { ...card, section: getSectionId(card.section) }
    const was = screen.cards[at]
    if (!alongside || !was || was.mark !== '' || card.mark === '') return withSection
    const same =
      was.stencilLink === card.stencilLink &&
      was.preamble === card.preamble &&
      seat(screen, was.section) === seat(readItem, card.section) &&
      JSON.stringify(was.values) === JSON.stringify(card.values)
    return same ? { ...withSection, id: was.id } : withSection
  })

  return { ...readItem, sections, cards }
}

/**
 * Whether two listings name the same stencils, in the same order and with the
 * same fields.
 */
export const areOffersEqual = (
  one: readonly StencilSummary[],
  other: readonly StencilSummary[],
): boolean =>
  one.length === other.length &&
  one.every((offer, at) => {
    const otherVal = other[at]
    return (
      // Stryker disable next-line ConditionalExpression: array lengths are equal so element at index is never undefined.
      otherVal !== undefined &&
      offer.path === otherVal.path &&
      offer.title === otherVal.title &&
      offer.fields.length === otherVal.fields.length &&
      offer.fields.every((field, seat) => field === otherVal.fields[seat])
    )
  })

/** Where the stencil of that title is filed, and nowhere where none is. */
export const findStencilPath = (offers: readonly StencilSummary[], name: string): string =>
  offers.find((offer) => offer.title === name)?.path ?? ''
