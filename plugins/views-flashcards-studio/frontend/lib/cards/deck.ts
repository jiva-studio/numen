/**
 * What a deck is, as plain values: cards holding the text typed into the slots
 * a card model names, grouped under their sections. Interpretation of any of it
 * is the caller's. No DOM, no measurement, no clock.
 */

import type { FieldValue } from './card'
import { getDeclaredFields, getSealedMap, type Problems, type InsertionPoint } from './order'

/**
 * Where a card dropped at the head of the deck lands: before the first section,
 * among the cards standing under none. The deck keeps this identity for itself,
 * and no card and no section may carry it.
 */
export const HEAD = 'the head of the deck'

/** What the end of a run is named by, which no card and no section may carry. */
const END = 'the end of '

/**
 * Where a card dropped past the last of a run lands: after everything standing
 * under that heading, and under the heading itself. The run before the first
 * section is named by the head of the deck.
 */
export const getRunEndPoint = (run: string): InsertionPoint => `${END}${run}`

/** Which run's end a drop target is, and nothing for a drop target that is not one. */
export const getRunEnd = (target: InsertionPoint): string | null =>
  typeof target === 'string' && target.startsWith(END) ? target.slice(END.length) : null

/** One card as the deck draws it. */
export interface DeckCard {
  readonly id: string
  readonly section: string | null
  readonly cardModel: string | null
  readonly filled: readonly FieldValue[]
}

/**
 * One section of a deck: a name and an identity assigned by the caller. A
 * section carries no card id and its name need not be unique, so nothing in the
 * file distinguishes one section from another.
 */
export interface DeckSection {
  readonly id: string
  readonly name: string
}

/** The words one card is drawn with, declared once. */
export interface CardWords {
  readonly remove: string
  readonly drag: string
  readonly cut: string
  readonly cardStem: string
  readonly nothing: string
  readonly unknown: (cardModel: string | null) => string
  readonly wrong: string
}

/**
 * The words a deck is drawn with: a card's, and what the two things it is added
 * to ask for.
 */
export interface DeckWords extends CardWords {
  readonly add: string
  readonly addSection: string
  readonly sectionStem: string
  readonly sectionObjection: string
}

export const DECK_WORDS: DeckWords = {
  add: 'Add a card',
  addSection: 'Add a section',
  remove: 'Remove',
  drag: 'Reorder',
  cut: 'Card model',
  cardStem: 'Card',
  sectionStem: 'Section',
  nothing: 'Nothing in it',
  unknown: (cardModel) =>
    cardModel === null ? 'Cut by no card model' : `No card model called ${cardModel}`,
  sectionObjection: 'A section needs a name',
  wrong: 'What is wrong',
}

/**
 * The problems the caller found in the deck it supplied. A card's problems are
 * placed under its heading and a value's under that value, so each message
 * appears next to what it concerns.
 */
export interface DeckProblems {
  readonly at: Problems
  readonly under: ReadonlyMap<string, Problems>
}

/** Nothing wrong with anything. */
export const NO_DECK_PROBLEMS: DeckProblems = Object.freeze({
  at: getSealedMap<string, readonly string[]>(),
  under: getSealedMap<string, Problems>(),
})

/** One value of a card, laid out under the card model that cuts it. */
export interface CardFieldValue extends FieldValue {
  readonly isDeclared: boolean
}

/**
 * A card's values in the order the card model requests them, empty where the card
 * omits a slot, and a slot the card writes twice appearing twice. A slot
 * the card model names twice is one slot and appears once. The values the card model
 * has no slot for come after the rest, marked as unnamed, and how
 * they are rendered is the caller's.
 */
export function getCardFieldValues(
  values: readonly FieldValue[],
  fields: readonly string[],
): readonly CardFieldValue[] {
  const status = getDeclaredFields(fields).flatMap((field) => {
    const content = values.filter((each) => each.field === field)
    if (!content.length) return [{ field, text: '', isDeclared: true }]
    return content.map((each) => ({ field, text: each.text, isDeclared: true }))
  })
  const stray = values
    .filter((each) => !fields.includes(each.field))
    .map((each) => ({ field: each.field, text: each.text, isDeclared: false }))
  return [...status, ...stray]
}

/** The empty values for a card model's slots, for a card with no typed text. */
export const getBlanks = (fields: readonly string[]): readonly FieldValue[] =>
  fields.map((field) => ({ field, text: '' }))
