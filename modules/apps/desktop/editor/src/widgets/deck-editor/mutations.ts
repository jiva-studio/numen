/**
 * Pure mutation operations for deck cards and sections.
 */
import {
  CARD_HEAD,
  cardEnded,
  type InsertionPoint,
} from '@numen/ui'
import type { Value } from '../../shared/flashcards/cards'
import { generateId, type IdMaker } from '../../shared/flashcards/identity'
import { nameOf } from '../../shared/paths'
import type { BufferCard, BufferDeck } from './types'

/**
 * A card made at the end of a section, cut by the stencil filed at a path, with
 * a value standing empty for each field. Under no section it is made at the end
 * of the cards standing before the first of them.
 */
export const addCard = (
  deck: BufferDeck,
  title: string,
  stencilPath: string,
  values: readonly Value[],
  section: string | null = null,
  mint: IdMaker = generateId,
): BufferDeck => {
  const ranks = new Map(deck.sections.map((each, index) => [each.id, index]))
  const rank = (id: string | null): number => (id === null ? -1 : (ranks.get(id) ?? -1))
  const mine = rank(section)

  let at = 0
  deck.cards.forEach((card, index) => {
    if (rank(card.section) <= mine) at = index + 1
  })

  return {
    ...deck,
    cards: [
      ...deck.cards.slice(0, at),
      {
        id: mint(),
        mark: '',
        section: section !== null && ranks.has(section) ? section : null,
        heading: '',
        stencilLink: stencilPath ? nameOf(stencilPath) : title,
        stencilPath,
        preamble: '',
        values,
      },
      ...deck.cards.slice(at),
    ],
  }
}

/** A card taken out of the deck. */
export const removeCard = (deck: BufferDeck, id: string): BufferDeck => ({
  ...deck,
  cards: deck.cards.filter((card) => card.id !== id),
})

/**
 * A card let go somewhere in the deck.
 */
export const dropCard = (deck: BufferDeck, id: string, at: InsertionPoint): BufferDeck => {
  const held = deck.cards.find((card) => card.id === id)
  if (!held) return deck
  const left = deck.cards.filter((card) => card.id !== id)

  if (at === CARD_HEAD) return { ...deck, cards: [{ ...held, section: null }, ...left] }

  const before = at === null ? -1 : left.findIndex((card) => card.id === at)
  if (before !== -1) {
    const under = { ...held, section: left[before]?.section ?? null }
    return { ...deck, cards: [...left.slice(0, before), under, ...left.slice(before)] }
  }

  if (at === null) {
    return { ...deck, cards: [...left, { ...held, section: deck.sections.at(-1)?.id ?? null }] }
  }

  const rank = (section: string | null): number =>
    section === null ? 0 : deck.sections.findIndex((each) => each.id === section) + 1
  const head = (section: string | null): number => {
    const seat = left.findIndex((card) => rank(card.section) >= rank(section))
    return seat === -1 ? left.length : seat
  }
  const put = (section: string | null, where: number): BufferDeck => ({
    ...deck,
    cards: [...left.slice(0, where), { ...held, section }, ...left.slice(where)],
  })

  const under = (card: BufferCard): string | null =>
    deck.sections.some((each) => each.id === card.section) ? card.section : null

  const last = (section: string | null): number => {
    let seat = -1
    left.forEach((card, index) => {
      if (under(card) === section) seat = index
    })
    return seat
  }

  const run = cardEnded(at)
  if (run !== null) {
    const section = run === CARD_HEAD ? null : run
    if (section !== null && !deck.sections.some((each) => each.id === section)) return deck
    const seat = last(section)
    return put(section, seat === -1 ? head(section) : seat + 1)
  }

  if (!deck.sections.some((section) => section.id === at)) return deck
  return put(at, head(at))
}

/** A section made at the end of the deck, holding no card. */
export const addSection = (deck: BufferDeck, name: string, mint: IdMaker = generateId): BufferDeck => ({
  ...deck,
  sections: [...deck.sections, { id: mint(), name, preamble: '' }],
})

/** A section under another name. */
export const renameSection = (deck: BufferDeck, id: string, name: string): BufferDeck => ({
  ...deck,
  sections: deck.sections.map((section) =>
    section.id === id ? { ...section, name } : section,
  ),
})

/** One piece of a deck's prose after another, with a line between the two. */
const after = (above: string, below: string): string =>
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
    preamble: above ? deck.preamble : after(deck.preamble, going?.preamble ?? ''),
    sections: deck.sections.flatMap((section) => {
      if (section.id === id) return []
      if (above && section.id === above.id) {
        return [{ ...section, preamble: after(section.preamble, going?.preamble ?? '') }]
      }
      return [section]
    }),
    cards: deck.cards.map((card) =>
      card.section === id ? { ...card, section: above?.id ?? null } : card,
    ),
  }
}

/**
 * One value of one card as it now reads.
 */
export const fillCard = (
  deck: BufferDeck,
  id: string,
  field: string,
  nth: number,
  text: string,
): BufferDeck => ({
  ...deck,
  cards: deck.cards.map((card) => {
    if (card.id !== id) return card

    let under = 0
    let found = false
    const values = card.values.map((value) => {
      if (value.field !== field) return value
      under += 1
      if (under !== nth) return value
      found = true
      return { field, text }
    })
    if (found) return { ...card, values }
    return text === '' ? card : { ...card, values: [...card.values, { field, text }] }
  }),
})
