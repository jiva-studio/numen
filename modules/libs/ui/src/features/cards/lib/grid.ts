/** The grid a deck is drawn as: its cards laid out as tiles under the sections they stand in. */

import type { Stencil } from './card'
import { getDeclaredFields, type InsertionPoint } from './order'
import {
  getCardFieldValues,
  getRunEnd,
  HEAD,
  type CardFieldValue,
  type DeckCard,
  type DeckSection,
} from './deck'

/** One value of a card as its tile draws it. */
export interface PlacedFieldValue extends CardFieldValue {
  /** at its current position among the values, counting from one. */
  readonly at: number
  /**
   * What tells it from every other value of its tile. Values standing under
   * one field are told apart by their order under it.
   */
  readonly key: string
  /** at its current position among the values the card writes under its own field, from one. */
  readonly index: number
  /**
   * It is the last box standing for its field, which is where what is wrong
   * with that field is said, once.
   */
  readonly isLast: boolean
}

/** One card as a tile of the grid. */
export interface Tile {
  /** What tells the card from every other of the deck, as the caller named it. */
  readonly id: string
  /** The section it is placed under, and nothing for a card before the first. */
  readonly section: string | null
  readonly stencil: string | null
  /** Its values, in the order its stencil asks for them. */
  readonly filled: readonly PlacedFieldValue[]
  /** at its current position among the tiles, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many stand in the grid with it, the plus among them. */
  readonly of: number
  /** The stencil it names is among the ones handed in. */
  readonly isKnown: boolean
  /** It is on its way somewhere else in the order. */
  readonly isDragged: boolean
}

/** One section as the grid draws it: the section, and at its current position. */
export interface PlacedSection extends DeckSection {
  /** at its current position among the sections, counting from one. */
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
   * It draws a drop target of its own. A run under a section is landed on by that
   * section's heading and a run holding cards by the first of them; the cards
   * before the first section have neither where none of them stands.
   */
  readonly isDropTarget: boolean
  /**
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
interface PlacementContext {
  readonly sectioned: ReadonlySet<string | null>
  readonly dragCard: string | null
}

function createTile(
  card: DeckCard,
  stencils: readonly Stencil[],
  placement: PlacementContext,
): Tile {
  const matchingStencil = stencils.find((each) => each.name === card.stencil)
  const fields = getDeclaredFields(matchingStencil?.fields ?? [])

  /** How many values the card writes under each field, as they are counted off. */
  const underItem = new Map<string, number>()
  const countOff = (field: string): number => {
    const index = (underItem.get(field) ?? 0) + 1
    underItem.set(field, index)
    return index
  }

  // A value the stencil does not name is the person's and stays in the file,
  // and nothing here draws it or says a word about it. A card whose stencil
  // was not handed in draws no value at all, and says which stencil it is
  // waiting for. A card naming no stencil waits for nothing, so everything it
  // wrote stands as it was written.
  const named = card.stencil !== null
  const counted = getCardFieldValues(card.filled, fields)
    .filter((each) => each.isDeclared || !named)
    .map((each, place) => {
      const index = countOff(each.field)
      return { ...each, at: place + 1, index, key: `${each.field}#${index}` }
    })

  return {
    id: card.id,
    section: placement.sectioned.has(card.section) ? card.section : null,
    stencil: card.stencil,
    filled: counted.map((each) => ({
      ...each,
      isLast: each.index === (underItem.get(each.field) ?? 0),
    })),
    at: 0,
    of: 0,
    isKnown: matchingStencil !== undefined,
    isDragged: card.id === placement.dragCard,
  }
}

/** Source data required to lay out a deck's grid. */
export interface DeckGridSource {
  readonly cards: readonly DeckCard[]
  readonly sections: readonly DeckSection[]
  readonly stencils: readonly Stencil[]
}

/**
 * The tiles of a deck, in the order the cards were handed in, each laid out
 * under the stencil it names and standing in the run of the section it is
 * under. A card naming a stencil that was not handed in keeps every value it
 * has, and a card naming a section that was not handed in stands before the
 * first section, where every card handed in is drawn and counted.
 */
export function getGrid(deck: DeckGridSource, dragCard: string | null = null): Grid {
  const { cards, sections, stencils } = deck
  const sectioned = new Set<string | null>(sections.map((section) => section.id))
  const placement: PlacementContext = { sectioned, dragCard }
  const tiles = cards.map((card) => createTile(card, stencils, placement))

  const tilesUnder = (section: string | null): readonly Tile[] =>
    tiles.filter((tile) => tile.section === section)

  const head = tilesUnder(null)
  const renderedTiles = [
    {
      id: HEAD,
      section: null,
      tiles: head,
      isDropTarget: head.length === 0 && sections.length > 0,
    },
    ...sections.map((section, index) => ({
      id: section.id,
      section: { id: section.id, name: section.name, at: index + 1 },
      tiles: tilesUnder(section.id),
      isDropTarget: false,
    })),
  ]

  // Everything the grid draws is counted off in the order it is drawn in, the
  // tiles of each run and then the plus that ends it, so a reader is told
  // where each of them stands among them all.
  let seat = 0
  const runs: readonly Run[] = renderedTiles.map((run) => {
    const arrangedTiles = run.tiles.map((tile) => ({ ...tile, at: ++seat }))
    const plus = run.section !== null || arrangedTiles.length > 0 || renderedTiles.length === 1
    return { ...run, tiles: arrangedTiles, plusAt: plus ? ++seat : null }
  })
  const of = seat

  return {
    runs: runs.map((run) => ({ ...run, tiles: run.tiles.map((tile) => ({ ...tile, of })) })),
    of,
  }
}

/**
 * Whether letting a dragged card go there moves it. A card let go where it
 * stands moves nothing: the head of the deck is where the first card standing
 * under no section already is, and the end of a run is where its last card is.
 */
export const isMoved = (runs: readonly Run[], dragCard: string, target: InsertionPoint): boolean => {
  if (target === dragCard) return false
  if (target === HEAD) return runs[0]?.tiles[0]?.id !== dragCard

  const run = getRunEnd(target)
  return runs.find((each) => each.id === run)?.tiles.at(-1)?.id !== dragCard
}
