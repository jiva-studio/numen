/**
 * What a deck is, as plain values: cards holding the text typed into the slots
 * a card model names, grouped under their sections. Interpretation of any of it
 * is the caller's. No DOM, no measurement, no clock.
 */

import type { FieldValue } from './card'
import { getDeclaredFields, createSealedMap, type Problems, type InsertionPoint } from './order'

/**
 * Where a card let go at the head of the deck lands: before the first section,
 * among the cards standing under none. The deck keeps this identity for itself,
 * and no card and no section may carry it.
 */
export const HEAD = 'the head of the deck'

/** What the end of a run is named by, which no card and no section may carry. */
const END = 'the end of '

/**
 * Where a card let go past the last of a run lands: after everything standing
 * under that heading, and under the heading itself. The run before the first
 * section is named by the head of the deck.
 */
export const createRunEnd = (run: string): InsertionPoint => `${END}${run}`

/** Which run's end a drop target is, and nothing for a drop target that is not one. */
export const getRunEnd = (target: InsertionPoint): string | null =>
  typeof target === 'string' && target.startsWith(END) ? target.slice(END.length) : null

/** One card as the deck draws it. */
export interface DeckCard {
  /**
   * Distinguishes this card from every other card of the deck while it is
   * displayed. It is the caller's identifier for the card and is returned in
   * every event about it, so no two cards may share one.
   */
  readonly id: string
  /** The section it is placed under, and nothing for a card before the first. */
  readonly section: string | null
  /** What the card is cut by, as a word to show, and nothing where nothing cuts it. */
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
  /** The text for a card with no values. */
  readonly nothing: string
  /**
   * The text for a card whose card model was not supplied, and for a card that
   * names none; the strip displays it in place of a card model's name.
   */
  readonly unknown: (cardModel: string | null) => string
  /** What a list of things wrong is called to a reader. */
  readonly wrong: string
}

/**
 * The words a deck is drawn with: a card's, and what the two things it is added
 * to ask for.
 */
export interface DeckWords extends CardWords {
  readonly add: string
  readonly addSection: string
  /** The base of a generated section name, followed by its position among the sections. */
  readonly sectionStem: string
  /**
   * The message for an unusable section name. Two sections may share a name, so
   * only an empty name is unusable.
   */
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
  /** What is wrong with each card, under the identity it was drawn by. */
  readonly at: Problems
  /**
   * What is wrong with one value of a card, under that card's identity and then
   */
  readonly under: ReadonlyMap<string, Problems>
}

/** Nothing wrong with anything. */
export const NO_DECK_PROBLEMS: DeckProblems = Object.freeze({
  at: createSealedMap<string, readonly string[]>(),
  under: createSealedMap<string, Problems>(),
})

/** One value of a card, laid out under the card model that cuts it. */
export interface CardFieldValue extends FieldValue {
  /** The card model names this slot. */
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
