/**
 * UI representation mappings and comparisons for decks and stencils.
 */
import type { DeckCard, DeckSection, Stencil } from '@numen/ui'
import type { StencilSummary, VaultCard } from '../../entities/deck/cards'
import type { BufferDeck } from './types'
import { cardsOf, sectionsOf } from './serialize'

export const drawnSectionsOf = (deck: BufferDeck): readonly DeckSection[] =>
  deck.sections.map(({ id, name }) => ({ id, name }))

/**
 * The cards as the grid draws them, each under the stencil its wikilink resolves to.
 */
export const drawnOf = (deck: BufferDeck, offers: readonly StencilSummary[]): readonly DeckCard[] => {
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
export const stencilsOf = (offers: readonly StencilSummary[]): readonly Stencil[] => {
  const taken = new Set<string>()
  const stencils: Stencil[] = []
  for (const offer of offers) {
    if (!offer.title || taken.has(offer.title)) continue
    taken.add(offer.title)
    stencils.push({ name: offer.title, fields: offer.fields })
  }
  return stencils
}

/** The cards as somebody wrote them, without the heading a write reads back. */
const written = (deck: BufferDeck): readonly Omit<VaultCard, 'heading'>[] =>
  cardsOf(deck).map(({ heading: _heading, ...card }) => card)

/**
 * Whether two decks read the same.
 */
export const sameDeck = (one: BufferDeck, other: BufferDeck): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(sectionsOf(one)) === JSON.stringify(sectionsOf(other)) &&
  JSON.stringify(written(one)) === JSON.stringify(written(other))

/**
 * The deck on screen under the headings the file now carries.
 */
export const applyHead = (held: BufferDeck, read: BufferDeck): BufferDeck => {
  const carried = (at: number): string => read.cards[at]?.heading ?? ''
  if (held.cards.every((card, at) => card.heading === carried(at))) return held
  return { ...held, cards: held.cards.map((card, at) => ({ ...card, heading: carried(at) })) }
}

/**
 * A reading of a deck under the identities the window already drew it by.
 */
export const applyName = (held: BufferDeck, read: BufferDeck): BufferDeck => {
  const seat = (deck: BufferDeck, section: string | null): number =>
    section === null ? -1 : deck.sections.findIndex((each) => each.id === section)

  const sections =
    held.sections.length === read.sections.length
      ? read.sections.map((section, at) => {
          const was = held.sections[at]
          return was && was.name === section.name && was.preamble === section.preamble
            ? { ...section, id: was.id }
            : section
        })
      : read.sections

  const under = (section: string | null): string | null =>
    section === null ? null : (sections[seat(read, section)]?.id ?? null)

  const alongside = held.cards.length === read.cards.length

  const cards = read.cards.map((card, at) => {
    const stands = { ...card, section: under(card.section) }
    const was = held.cards[at]
    if (!alongside || !was || was.mark !== '' || card.mark === '') return stands
    const same =
      was.stencilLink === card.stencilLink &&
      was.preamble === card.preamble &&
      seat(held, was.section) === seat(read, card.section) &&
      JSON.stringify(was.values) === JSON.stringify(card.values)
    return same ? { ...stands, id: was.id } : stands
  })

  return { ...read, sections, cards }
}

/**
 * Whether two listings name the same stencils, in the same order and with the
 * same fields.
 */
export const sameOffers = (one: readonly StencilSummary[], other: readonly StencilSummary[]): boolean =>
  one.length === other.length &&
  one.every((offer, at) => {
    const against = other[at]
    return (
      against !== undefined &&
      offer.path === against.path &&
      offer.title === against.title &&
      offer.fields.length === against.fields.length &&
      offer.fields.every((field, seat) => field === against.fields[seat])
    )
  })

/** Where the stencil of that title is filed, and nowhere where none is. */
export const pathOfCut = (offers: readonly StencilSummary[], name: string): string =>
  offers.find((offer) => offer.title === name)?.path ?? ''
