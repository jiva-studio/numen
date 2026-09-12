/**
 * What a deck is, as plain values: cards standing what a person typed in the
 * slots a stencil names, under the sections they stand in. What any of it means
 * is the caller's. No DOM, no measurement, no clock.
 */

import type { FieldValue } from './card'
import {
  getDeclaredFields,
  createSealedMap,
  type Problems,
  type InsertionPoint,
} from './order'

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
export const endOf = (run: string): InsertionPoint => `${END}${run}`

/** Which run's end a landing is, and nothing for a landing that is not one. */
export const getRunEnd = (at: InsertionPoint): string | null =>
  typeof at === 'string' && at.startsWith(END) ? at.slice(END.length) : null

/** One card as the deck draws it. */
export interface DeckCard {
  /**
   * What tells this card from every other of the deck, for as long as it is
   * drawn. It is the caller's own word for the card and travels back in every
   * event about it, so no two cards may carry one.
   */
  readonly id: string
  /** The section it stands under, and nothing for a card before the first. */
  readonly section: string | null
  /** What the card is cut by, as a word to show, and nothing where nothing cuts it. */
  readonly stencil: string | null
  readonly filled: readonly FieldValue[]
}

/**
 * One section of a deck: a name, under an identity the caller minted for it. A
 * section carries no mark and its name need not be unique, so nothing the file
 * holds tells one from another.
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
  /** What a card is announced by, before the place it stands in the deck. */
  readonly cardStem: string
  /** What is said of a card holding no value at all. */
  readonly nothing: string
  /**
   * What is said of a card cut by a stencil that was not handed in, and of a
   * card that names none, which the strip says in place of a stencil's name.
   */
  readonly unknown: (stencil: string | null) => string
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
  /** What a section is named from, before the place it stands among them. */
  readonly sectionStem: string
  /**
   * What is said of a section's name that cannot be used. Two sections may
   * carry one name, so a name with nothing in it is the whole of it.
   */
  readonly sectionObjection: string
}

export const DECK_WORDS: DeckWords = {
  add: 'Add a card',
  addSection: 'Add a section',
  remove: 'Remove',
  drag: 'Reorder',
  cut: 'Stencil',
  cardStem: 'Card',
  sectionStem: 'Section',
  nothing: 'Nothing in it',
  unknown: (stencil) => (stencil === null ? 'Cut by no stencil' : `No stencil called ${stencil}`),
  sectionObjection: 'A section needs a name',
  wrong: 'What is wrong',
}

/**
 * What the caller found wrong with what it handed in. A card's stands under its
 * heading and a value's stands under that value, so nothing is said in a place
 * that leaves a person guessing what it is about.
 */
export interface Wrong {
  /** What is wrong with each card, under the identity it was drawn by. */
  readonly at: Problems
  /**
   * What is wrong with one value of a card, under that card's identity and then
   * the field the value stands in.
   */
  readonly under: ReadonlyMap<string, Problems>
}

/** Nothing wrong with anything. */
export const NOTHING_WRONG: Wrong = Object.freeze({
  at: createSealedMap<string, readonly string[]>(),
  under: createSealedMap<string, Problems>(),
})

/** One value of a card, laid out under the stencil that cuts it. */
export interface CardFieldValue extends FieldValue {
  /** The stencil names this slot. */
  readonly declared: boolean
}

/**
 * A card's values in the order the stencil asks for them, empty where the card
 * leaves a slot out, and a slot the card writes twice standing twice. A slot
 * the stencil names twice is one slot and stands once. The values the stencil
 * names nothing for come after the rest, marked as named by nothing, and what
 * is drawn of them is the caller's.
 */
export function getCardFieldValues(
  values: readonly FieldValue[],
  fields: readonly string[],
): readonly CardFieldValue[] {
  const stood = getDeclaredFields(fields).flatMap((field) => {
    const written = values.filter((each) => each.field === field)
    if (!written.length) return [{ field, text: '', declared: true }]
    return written.map((each) => ({ field, text: each.text, declared: true }))
  })
  const stray = values
    .filter((each) => !fields.includes(each.field))
    .map((each) => ({ field: each.field, text: each.text, declared: false }))
  return [...stood, ...stray]
}

/** The empty values a stencil's slots make, for a card nobody has typed into. */
export const blanks = (fields: readonly string[]): readonly FieldValue[] =>
  fields.map((field) => ({ field, text: '' }))
