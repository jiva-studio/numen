/**
 * What a deck is, as plain values: cards standing what a person typed in the
 * slots a stencil names, laid out as a grid of tiles. What any of it means is
 * the caller's. No DOM, no measurement, no clock.
 */

import { declared, type Against } from './order'
import type { Cut } from './stencil'

/** One named slot and what stands in it. */
export interface Filled {
  readonly field: string
  readonly text: string
}

/** One card as the deck draws it. */
export interface Drawn {
  readonly id: string
  readonly name: string
  /** What the card is cut by, as a word to show, and nothing where nothing cuts it. */
  readonly stencil: string | null
  readonly filled: readonly Filled[]
}

/** The words one card is drawn with, declared once. */
export interface CardWords {
  readonly remove: string
  readonly carry: string
  readonly cut: string
  readonly name: string
  /** What is said of a card holding no value at all. */
  readonly nothing: string
  /** What is said of a card whose stencil the vault does not hold. */
  readonly unknown: (stencil: string | null) => string
  /** What is said of a value standing in the field the card is named by. */
  readonly twice: string
  /** What a list of things wrong is called to a reader. */
  readonly wrong: string
}

/** The words a deck is drawn with: a card's, and the two the plus asks for. */
export interface DeckWords extends CardWords {
  readonly add: string
  readonly cardStem: string
}

export const DECK_WORDS: DeckWords = {
  add: 'Add a card',
  remove: 'Remove',
  carry: 'Reorder',
  cut: 'Stencil',
  name: 'Name',
  nothing: 'Nothing in it',
  unknown: (stencil) => (stencil === null ? 'Cut by no stencil' : `No stencil called ${stencil}`),
  twice: 'The card is named by this field',
  wrong: 'What is wrong',
  cardStem: 'Card',
}

/**
 * What the caller found wrong with what it handed in. A card's stands under its
 * heading and a value's stands under that value, so nothing is said in a place
 * that leaves a person guessing what it is about.
 */
export interface Wrong {
  /** What is wrong with each card, under the identity it was drawn by. */
  readonly at: Against
  /**
   * What is wrong with one value of a card, under that card's identity and then
   * the field the value stands in.
   */
  readonly under: ReadonlyMap<string, Against>
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
  under: sealed<string, Against>(),
})

/** One value of a card, laid out under the stencil that cuts it. */
export interface Laid extends Filled {
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
export function laid(filled: readonly Filled[], fields: readonly string[]): readonly Laid[] {
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
export const blanks = (fields: readonly string[]): readonly Filled[] =>
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
  /**
   * Where it stands among the values the card writes under its own field, from
   * one. The name a card carries is written in its heading and in no value of
   * it, so the one that names the card stands among none of them.
   */
  readonly nth: number
  /** It is the field the card is named by, which is the stencil's first. */
  readonly names: boolean
  /**
   * It names the card and stands as a value as well. A card is named once, so
   * the value is kept and shown, and no face lays it out.
   */
  readonly twice: boolean
  /**
   * It is the last box standing for its field, which is where what is wrong
   * with that field is said, once.
   */
  readonly last: boolean
}

/** One card as a tile of the grid. */
export interface Tile {
  readonly id: string
  readonly name: string
  readonly stencil: string | null
  /** Its values, the field naming the card first, laid out under its stencil. */
  readonly filled: readonly Stood[]
  /** Where it stands among the tiles, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many stand in the grid with it, the plus among them. */
  readonly of: number
  /** The stencil it names is among the ones handed in. */
  readonly known: boolean
  /** A field of its stencil names it. Where none does, the tile says the name it was handed. */
  readonly named: boolean
  /** It is on its way somewhere else in the order. */
  readonly carried: boolean
}

/** The grid a deck draws: the cards as tiles, and where the plus stands. */
export interface Grid {
  readonly tiles: readonly Tile[]
  /** Where the plus stands, counting from one. It stands last. */
  readonly plusAt: number
  /** How many stand in the grid, the plus among them. */
  readonly of: number
}

/**
 * The tiles of a deck, in the order the cards were handed in, each laid out
 * under the stencil it names. A card naming a stencil that was not handed in
 * keeps every value it has.
 *
 * A stencil's first field names the card. Its value stands in the name the card
 * was handed and in none of its values, and it is put back at the head of them
 * so the tile draws it as it draws every other field.
 */
export function grid(
  cards: readonly Drawn[],
  cuts: readonly Cut[],
  carried: string | null,
): Grid {
  const of = cards.length + 1
  const tiles = cards.map((card, index) => {
    const cut = cuts.find((each) => each.name === card.stencil)
    const fields = declared(cut?.fields ?? [])
    const first = fields[0]

    const rest = laid(card.filled, fields.slice(1)).map((each) => ({
      ...each,
      names: false,
      twice: first !== undefined && each.field === first,
    }))
    const named = first === undefined ? [] : [
      { field: first, text: card.name, declared: true, names: true, twice: false },
    ]

    /** How many values the card writes under each field, as they are counted off. */
    const under = new Map<string, number>()
    const told = (field: string): number => {
      const nth = (under.get(field) ?? 0) + 1
      under.set(field, nth)
      return nth
    }

    // A value the stencil does not name is the person's and stays in the file,
    // and nothing here draws it or says a word about it. A card whose stencil
    // the vault does not hold draws no value at all, and says which stencil it
    // is waiting for.
    const counted = [...named, ...rest]
      .filter((each) => each.declared || each.twice)
      .map((each, place) => {
        const nth = each.names ? 0 : told(each.field)
        return { ...each, at: place + 1, nth, key: `${each.field}#${nth}` }
      })

    return {
      id: card.id,
      name: card.name,
      stencil: card.stencil,
      filled: counted.map((each) => ({
        ...each,
        last: each.nth === (under.get(each.field) ?? 0),
      })),
      at: index + 1,
      of,
      known: cut !== undefined,
      named: first !== undefined,
      carried: card.id === carried,
    }
  })
  return { tiles, plusAt: cards.length + 1, of }
}
