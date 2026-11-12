/**
 * UI representation mappings and comparisons for decks and card models.
 */
import type { DeckCard, DeckSection, CardModel } from '../../../lib/cards'
import type { BufferDeck } from './types'
import {
  serializeBufferCardsToVaultCards,
  serializeBufferSectionsToVaultSections,
} from './serialize'
import type { CardModelSummary, VaultCard } from '../../../types'

export const createDeckSections = (deck: BufferDeck): readonly DeckSection[] =>
  deck.sections.map(({ id, name }) => ({ id, name }))

/**
 * The cards as the grid draws them, each under the card model its wikilink resolves to.
 */
export const createDeckCards = (
  deck: BufferDeck,
  summaries: readonly CardModelSummary[],
): readonly DeckCard[] => {
  const titles = new Map(summaries.map((summary) => [summary.path, summary.title]))
  return deck.cards.map((card) => ({
    id: card.id,
    section: card.section,
    cardModel: (titles.get(card.cardModelPath) ?? card.cardModelLink) || null,
    filled: card.values.map((value) => ({ field: value.field, text: value.text })),
  }))
}

/**
 * The card models a card may be cut by, under the word a card names one by.
 */
export const createCardModelChoices = (
  summaries: readonly CardModelSummary[],
): readonly CardModel[] => {
  const seenTitles = new Set<string>()
  const cardModels: CardModel[] = []
  for (const summary of summaries) {
    if (!summary.title || seenTitles.has(summary.title)) continue
    seenTitles.add(summary.title)
    cardModels.push({ name: summary.title, fields: summary.fields })
  }
  return cardModels
}

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
      ? readItem.sections.map((section, index) => {
          const was = screen.sections[index]
          return was && was.name === section.name && was.preamble === section.preamble
            ? { ...section, id: was.id }
            : section
        })
      : readItem.sections

  // Stryker disable next-line ConditionalExpression: index -1 from seat yields null fallback either way.
  const getSectionId = (section: string | null): string | null =>
    section === null ? null : (sections[seat(readItem, section)]?.id ?? null)

  const alongside = screen.cards.length === readItem.cards.length

  const cards = readItem.cards.map((card, index) => {
    const withSection = { ...card, section: getSectionId(card.section) }
    const was = screen.cards[index]
    if (!alongside || !was || was.cardId !== '' || card.cardId === '') return withSection
    const same =
      was.cardModelLink === card.cardModelLink &&
      was.preamble === card.preamble &&
      seat(screen, was.section) === seat(readItem, card.section) &&
      JSON.stringify(was.values) === JSON.stringify(card.values)
    return same ? { ...withSection, id: was.id } : withSection
  })

  return { ...readItem, sections, cards }
}

/**
 * Whether two listings name the same card models, in the same order and with the
 * same fields.
 */
export const areCardModelChoicesEqual = (
  one: readonly CardModelSummary[],
  other: readonly CardModelSummary[],
): boolean =>
  one.length === other.length &&
  one.every((summary, index) => {
    const otherVal = other[index]
    return (
      // Stryker disable next-line ConditionalExpression: array lengths are equal so element at index is never undefined.
      otherVal !== undefined &&
      summary.path === otherVal.path &&
      summary.title === otherVal.title &&
      summary.fields.length === otherVal.fields.length &&
      summary.fields.every((field, seat) => field === otherVal.fields[seat])
    )
  })

/** Where the card model of that title is filed, and nowhere where none is. */
export const findCardModelPath = (summaries: readonly CardModelSummary[], name: string): string =>
  summaries.find((summary) => summary.title === name)?.path ?? ''
