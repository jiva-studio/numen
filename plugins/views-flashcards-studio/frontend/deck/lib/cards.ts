/**
 * Pure mutation operations for deck cards.
 */
import { HEAD as CARD_HEAD, getRunEnd, type InsertionPoint } from '../../ui/cards'
import { type Value, generateId, type IdMaker } from '../../cards'
import { getFileStem } from '@numen/plugin-sdk'
import type { BufferCard, BufferDeck } from '../types'

export interface AddCardOptions {
  cardModelPath: string
  values: readonly Value[]
  section?: string | null
  generateCardId?: IdMaker
}

/**
 * A card made at the end of a section, cut by the card model filed at a path, with
 * a value standing empty for each field. Under no section it is made at the end
 * of the cards standing before the first of them.
 */
export const addCard = (deck: BufferDeck, title: string, options: AddCardOptions): BufferDeck => {
  const { cardModelPath, values, section = null, generateCardId = generateId } = options
  const ranks = new Map<string | null, number>(deck.sections.map((each, index) => [each.id, index]))
  const rank = (id: string | null): number => ranks.get(id) ?? -1
  const mine = rank(section)

  let targetIndex = 0
  deck.cards.forEach((card, index) => {
    if (rank(card.section) <= mine) targetIndex = index + 1
  })

  return {
    ...deck,
    cards: [
      ...deck.cards.slice(0, targetIndex),
      {
        id: generateCardId(),
        cardId: '',
        section: ranks.has(section) ? section : null,
        heading: '',
        cardModelLink: cardModelPath ? getFileStem(cardModelPath) : title,
        cardModelPath,
        preamble: '',
        values,
      },
      ...deck.cards.slice(targetIndex),
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
export const dropCard = (deck: BufferDeck, id: string, target: InsertionPoint): BufferDeck => {
  const targetCard = deck.cards.find((card) => card.id === id)
  if (!targetCard) return deck
  const otherCards = deck.cards.filter((card) => card.id !== id)

  if (target === CARD_HEAD)
    return { ...deck, cards: [{ ...targetCard, section: null }, ...otherCards] }

  const before = otherCards.findIndex((card) => card.id === target)
  if (before !== -1) {
    // Stryker disable next-line OptionalChaining: the index was found in this array, so the card exists
    const underVal = { ...targetCard, section: otherCards[before]?.section ?? null }
    return {
      ...deck,
      cards: [...otherCards.slice(0, before), underVal, ...otherCards.slice(before)],
    }
  }

  if (target === null) {
    return { ...deck, cards: [...otherCards, { ...targetCard, section: getLastSection(deck) }] }
  }

  return dropOnSection(deck, { card: targetCard, left: otherCards, target })
}

interface DropOnSectionParams {
  card: BufferCard
  left: readonly BufferCard[]
  target: string
}

/**
 * A card let go on a section: after the last card standing under it, and where
 * the section begins while it holds none.
 */
const dropOnSection = (deck: BufferDeck, params: DropOnSectionParams): BufferDeck => {
  const { card, left, target } = params
  const rank = (section: string | null): number =>
    deck.sections.findIndex((each) => each.id === section)
  const head = (section: string | null): number => {
    const seat = left.findIndex((each) => rank(each.section) >= rank(section))
    return seat === -1 ? left.length : seat
  }
  const placeCard = (section: string | null, where: number): BufferDeck => ({
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

  const run = getRunEnd(target)
  if (run !== null) {
    const section = run === CARD_HEAD ? null : run
    if (section !== null && !deck.sections.some((each) => each.id === section)) return deck
    const seat = last(section)
    return placeCard(section, seat === -1 ? head(section) : seat + 1)
  }

  if (!deck.sections.some((section) => section.id === target)) return deck
  return placeCard(target, head(target))
}

export interface ReplaceValueTarget {
  field: string
  index: number
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
  const { field, index, text } = target
  return {
    ...deck,
    cards: deck.cards.map((card) => {
      if (card.id !== id) return card

      let underCount = 0
      let found = false
      const values = card.values.map((value) => {
        if (value.field !== field) return value
        underCount += 1
        if (underCount !== index) return value
        found = true
        return { field, text }
      })
      if (found) return { ...card, values }
      return text === '' ? card : { ...card, values: [...card.values, { field, text }] }
    }),
  }
}
