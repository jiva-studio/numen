/**
 * Pure mutation operations for deck cards.
 */
import { CARD_HEAD, getRunEnd, type InsertionPoint } from '@numen/ui'
import { type Value, generateId, type IdMaker } from '../../cards'
import { getFileStem } from '@numen/plugin-sdk'
import type { BufferCard, BufferDeck } from '../types'

export interface AddCardOptions {
  stencilPath: string
  values: readonly Value[]
  section?: string | null
  generateCardId?: IdMaker
}

/**
 * A card made at the end of a section, cut by the stencil filed at a path, with
 * a value standing empty for each field. Under no section it is made at the end
 * of the cards standing before the first of them.
 */
export const addCard = (deck: BufferDeck, title: string, options: AddCardOptions): BufferDeck => {
  const { stencilPath, values, section = null, generateCardId = generateId } = options
  const ranks = new Map<string | null, number>(deck.sections.map((each, index) => [each.id, index]))
  const rank = (id: string | null): number => ranks.get(id) ?? -1
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
        id: generateCardId(),
        mark: '',
        section: ranks.has(section) ? section : null,
        heading: '',
        stencilLink: stencilPath ? getFileStem(stencilPath) : title,
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

/** The section the cards at the end of the deck stand under. */
const getLastSection = (deck: BufferDeck): string | null => deck.sections.at(-1)?.id ?? null

/**
 * A card let go somewhere in the deck.
 */
export const dropCard = (deck: BufferDeck, id: string, at: InsertionPoint): BufferDeck => {
  const targetCard = deck.cards.find((card) => card.id === id)
  if (!targetCard) return deck
  const otherCards = deck.cards.filter((card) => card.id !== id)

  if (at === CARD_HEAD) return { ...deck, cards: [{ ...targetCard, section: null }, ...otherCards] }

  const before = otherCards.findIndex((card) => card.id === at)
  if (before !== -1) {
    // Stryker disable next-line OptionalChaining: the index was found in this array, so the card exists
    const underVal = { ...targetCard, section: otherCards[before]?.section ?? null }
    return { ...deck, cards: [...otherCards.slice(0, before), underVal, ...otherCards.slice(before)] }
  }

  if (at === null) {
    return { ...deck, cards: [...otherCards, { ...targetCard, section: getLastSection(deck) }] }
  }

  return dropOnSection(deck, { card: targetCard, left: otherCards, at })
}

interface DropOnSectionParams {
  card: BufferCard
  left: readonly BufferCard[]
  at: string
}

/**
 * A card let go on a section: after the last card standing under it, and where
 * the section begins while it holds none.
 */
const dropOnSection = (deck: BufferDeck, params: DropOnSectionParams): BufferDeck => {
  const { card, left, at } = params
  const rank = (section: string | null): number =>
    deck.sections.findIndex((each) => each.id === section)
  const head = (section: string | null): number => {
    const seat = left.findIndex((each) => rank(each.section) >= rank(section))
    return seat === -1 ? left.length : seat
  }
  const put = (section: string | null, where: number): BufferDeck => ({
    ...deck,
    cards: [...left.slice(0, where), { ...card, section }, ...left.slice(where)],
  })

  const getSection = (each: BufferCard): string | null =>
    deck.sections.some((section) => section.id === each.section) ? each.section : null

  const last = (section: string | null): number => {
    let seat = -1
    left.forEach((each, index) => {
      if (getSection(each) === section) seat = index
    })
    return seat
  }

  const run = getRunEnd(at)
  if (run !== null) {
    const section = run === CARD_HEAD ? null : run
    if (section !== null && !deck.sections.some((each) => each.id === section)) return deck
    const seat = last(section)
    return put(section, seat === -1 ? head(section) : seat + 1)
  }

  if (!deck.sections.some((section) => section.id === at)) return deck
  return put(at, head(at))
}

export interface ReplaceValueTarget {
  field: string
  nth: number
  text: string
}

/**
 * One value of one card as it now reads.
 */
export const replaceCardValue = (
  deck: BufferDeck,
  id: string,
  target: ReplaceValueTarget,
): BufferDeck => {
  const { field, nth, text } = target
  return {
    ...deck,
    cards: deck.cards.map((card) => {
      if (card.id !== id) return card

      let underCount = 0
      let found = false
      const values = card.values.map((value) => {
        if (value.field !== field) return value
        underCount += 1
        if (underCount !== nth) return value
        found = true
        return { field, text }
      })
      if (found) return { ...card, values }
      return text === '' ? card : { ...card, values: [...card.values, { field, text }] }
    }),
  }
}
