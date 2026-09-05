/**
 * What a deck is, as plain values: cards standing what a person typed in the
 * slots a stencil names, laid out as a grid of tiles under the sections they
 * stand in. What any of it means is the caller's. No DOM, no measurement, no
 * clock.
 */

import { declared, type Problems, type InsertionPoint } from './order'
import type { Stencil } from './stencil'

/** One named slot and what stands in it. */
export interface FieldValue {
  readonly field: string
  readonly text: string
}

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
export const ended = (at: InsertionPoint): string | null =>
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
  readonly carry: string
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
  carry: 'Reorder',
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

/**
 * A map of nothing that stays a map of nothing. An empty default stands for
 * every caller at once, so putting anything into it is refused.
 */
export function sealed<K, V>(): ReadonlyMap<K, V> {
  const empty = new Map<K, V>()
  const refuses = (): never => {
    throw new TypeError('an empty default holds nothing')
  }
  return Object.freeze(Object.assign(empty, { set: refuses, delete: refuses, clear: refuses }))
}

/** Nothing wrong with anything. */
export const NOTHING_WRONG: Wrong = Object.freeze({
  at: sealed<string, readonly string[]>(),
  under: sealed<string, Problems>(),
})

/** One value of a card, laid out under the stencil that cuts it. */
export interface Laid extends FieldValue {
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
export function laid(filled: readonly FieldValue[], fields: readonly string[]): readonly Laid[] {
  const stood = declared(fields).flatMap((field) => {
    const written = filled.filter((each) => each.field === field)
    if (!written.length) return [{ field, text: '', declared: true }]
    return written.map((each) => ({ field, text: each.text, declared: true }))
  })
  const stray = filled
    .filter((each) => !fields.includes(each.field))
    .map((each) => ({ field: each.field, text: each.text, declared: false }))
  return [...stood, ...stray]
}

/** The empty values a stencil's slots make, for a card nobody has typed into. */
export const blanks = (fields: readonly string[]): readonly FieldValue[] =>
  fields.map((field) => ({ field, text: '' }))

/** One value of a card as its tile draws it. */
export interface Stood extends Laid {
  /** Where it stands among the values, counting from one. */
  readonly at: number
  /**
   * What tells it from every other value of its tile. Values standing under
   * one field are told apart by their order under it.
   */
  readonly key: string
  /** Where it stands among the values the card writes under its own field, from one. */
  readonly nth: number
  /**
   * It is the last box standing for its field, which is where what is wrong
   * with that field is said, once.
   */
  readonly last: boolean
}

/** One card as a tile of the grid. */
export interface Tile {
  /** What tells the card from every other of the deck, as the caller named it. */
  readonly id: string
  /** The section it stands under, and nothing for a card before the first. */
  readonly section: string | null
  readonly stencil: string | null
  /** Its values, in the order its stencil asks for them. */
  readonly filled: readonly Stood[]
  /** Where it stands among the tiles, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many stand in the grid with it, the plus among them. */
  readonly of: number
  /** The stencil it names is among the ones handed in. */
  readonly known: boolean
  /** It is on its way somewhere else in the order. */
  readonly carried: boolean
}

/** One section as the grid draws it: the section, and where it stands. */
export interface PlacedSection extends DeckSection {
  /** Where it stands among the sections, counting from one. */
  readonly at: number
}

/** One run of a deck: a section, where the cards stand under one, and those cards. */
export interface Run {
  /** Where a card let go on the run itself lands, which is at the head of it. */
  readonly id: string
  /** The section they stand under, and nothing for the cards before the first. */
  readonly section: PlacedSection | null
  readonly tiles: readonly Tile[]
  /**
   * It draws a landing of its own. A run under a section is landed on by that
   * section's heading and a run holding cards by the first of them; the cards
   * before the first section have neither where none of them stands.
   */
  readonly landing: boolean
  /**
   * Where this run's plus stands in the grid, counting from one, and nothing
   * where the run draws none. A card is made at the end of a run, so a run
   * cards may be put in has one: every section, and the cards before the first
   * section wherever any of them stands there.
   */
  readonly plusAt: number | null
}

/** The grid a deck draws: its runs of cards, and how many stand in it. */
export interface Grid {
  /** The cards before the first section, and then one run for each section. */
  readonly runs: readonly Run[]
  /** How many stand in the grid, every plus among them. */
  readonly of: number
}

/**
 * The tiles of a deck, in the order the cards were handed in, each laid out
 * under the stencil it names and standing in the run of the section it is
 * under. A card naming a stencil that was not handed in keeps every value it
 * has, and a card naming a section that was not handed in stands before the
 * first section, where every card handed in is drawn and counted.
 */
export function grid(
  cards: readonly DeckCard[],
  sections: readonly DeckSection[],
  stencils: readonly Stencil[],
  carried: string | null,
): Grid {
  const sectioned = new Set(sections.map((section) => section.id))
  const tiles = cards.map((card) => {
    const cut = stencils.find((each) => each.name === card.stencil)
    const fields = declared(cut?.fields ?? [])

    /** How many values the card writes under each field, as they are counted off. */
    const under = new Map<string, number>()
    const told = (field: string): number => {
      const nth = (under.get(field) ?? 0) + 1
      under.set(field, nth)
      return nth
    }

    // A value the stencil does not name is the person's and stays in the file,
    // and nothing here draws it or says a word about it. A card whose stencil
    // was not handed in draws no value at all, and says which stencil it is
    // waiting for. A card naming no stencil waits for nothing, so everything it
    // wrote stands as it was written.
    const named = card.stencil !== null
    const counted = laid(card.filled, fields)
      .filter((each) => each.declared || !named)
      .map((each, place) => {
        const nth = told(each.field)
        return { ...each, at: place + 1, nth, key: `${each.field}#${nth}` }
      })

    return {
      id: card.id,
      section: card.section !== null && sectioned.has(card.section) ? card.section : null,
      stencil: card.stencil,
      filled: counted.map((each) => ({
        ...each,
        last: each.nth === (under.get(each.field) ?? 0),
      })),
      at: 0,
      of: 0,
      known: cut !== undefined,
      carried: card.id === carried,
    }
  })

  const tilesUnder = (section: string | null): readonly Tile[] =>
    tiles.filter((tile) => tile.section === section)

  const head = tilesUnder(null)
  const drawn = [
    { id: HEAD, section: null, tiles: head, landing: head.length === 0 && sections.length > 0 },
    ...sections.map((section, index) => ({
      id: section.id,
      section: { id: section.id, name: section.name, at: index + 1 },
      tiles: tilesUnder(section.id),
      landing: false,
    })),
  ]

  // Everything the grid draws is counted off in the order it is drawn in, the
  // tiles of each run and then the plus that ends it, so a reader is told
  // where each of them stands among them all.
  let seat = 0
  const runs: readonly Run[] = drawn.map((run) => {
    const laid = run.tiles.map((tile) => ({ ...tile, at: ++seat }))
    const plus = run.section !== null || laid.length > 0 || drawn.length === 1
    return { ...run, tiles: laid, plusAt: plus ? ++seat : null }
  })
  const of = seat

  return {
    runs: runs.map((run) => ({ ...run, tiles: run.tiles.map((tile) => ({ ...tile, of })) })),
    of,
  }
}

/**
 * Whether letting a carried card go there moves it. A card let go where it
 * stands moves nothing: the head of the deck is where the first card standing
 * under no section already is, and the end of a run is where its last card is.
 */
export const lands = (runs: readonly Run[], carried: string, at: InsertionPoint): boolean => {
  if (at === carried) return false
  if (at === HEAD) return runs[0]?.tiles[0]?.id !== carried

  const run = ended(at)
  if (run !== null) return runs.find((each) => each.id === run)?.tiles.at(-1)?.id !== carried
  return true
}
