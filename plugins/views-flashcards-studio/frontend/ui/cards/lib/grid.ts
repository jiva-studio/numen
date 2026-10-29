/** The grid a deck is drawn as: its cards laid out as tiles under the sections they stand in. */

import type { CardModel } from './card'
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
   * Identifies the value within its tile. Values under one field are
   * distinguished by their order under it.
   */
  readonly key: string
  /** at its current position among the values the card writes under its own field, from one. */
  readonly index: number
  /**
   * Whether this is the last box of its field, where the field's error is shown
   * once.
   */
  readonly isLast: boolean
}

/** One card as a tile of the grid. */
export interface Tile {
  /** Identifies the card within the deck, as named by the caller. */
  readonly id: string
  /** The section it is placed under, and nothing for a card before the first. */
  readonly section: string | null
  readonly cardModel: string | null
  /** Its values, in the order its card model declares them. */
  readonly filled: readonly PlacedFieldValue[]
  /** at its current position among the tiles, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many stand in the grid with it, the plus among them. */
  readonly of: number
  /** The card model it names is among the ones handed in. */
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
   * Whether the run draws a drop target of its own. A run under a section is
   * targeted by that section's heading and a run holding cards by the first of
   * them; the cards before the first section have neither when they hold no card.
   */
  readonly isDropTarget: boolean
  /**
   * Position of the run's plus among the grid positions, and null when the run
   * draws none. A card is created at the end of a run, so every run that can
   * receive cards has one: every section, and the cards before the first section
   * when they hold a card or the deck has no section.
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
 * Inputs shared by every tile: the ids of the sections that exist, and the id of
 * the card being dragged.
 */
interface PlacementContext {
  readonly sectioned: ReadonlySet<string | null>
  readonly dragCard: string | null
}

function createTile(
  card: DeckCard,
  cardModels: readonly CardModel[],
  placement: PlacementContext,
): Tile {
  const matchingCardModel = cardModels.find((each) => each.name === card.cardModel)
  const fields = getDeclaredFields(matchingCardModel?.fields ?? [])

  /** How many values the card writes under each field, as they are counted off. */
  const underItem = new Map<string, number>()
  const countOff = (field: string): number => {
    const index = (underItem.get(field) ?? 0) + 1
    underItem.set(field, index)
    return index
  }

  // A value the card model does not declare stays in the file and is not drawn. A
  // card whose card model was not supplied draws no value and is not known. A card
  // naming no card model draws every value it contains.
  const named = card.cardModel !== null
  const counted = getCardFieldValues(card.filled, fields)
    .filter((each) => each.isDeclared || !named)
    .map((each, place) => {
      const index = countOff(each.field)
      return { ...each, at: place + 1, index, key: `${each.field}#${index}` }
    })

  return {
    id: card.id,
    section: placement.sectioned.has(card.section) ? card.section : null,
    cardModel: card.cardModel,
    filled: counted.map((each) => ({
      ...each,
      isLast: each.index === (underItem.get(each.field) ?? 0),
    })),
    at: 0,
    of: 0,
    isKnown: matchingCardModel !== undefined,
    isDragged: card.id === placement.dragCard,
  }
}

/** Source data required to lay out a deck's grid. */
export interface DeckGridSource {
  readonly cards: readonly DeckCard[]
  readonly sections: readonly DeckSection[]
  readonly cardModels: readonly CardModel[]
}

/**
 * The grid of a deck: its cards in input order, each laid out under the card
 * model it names and placed in the run of its section. A card naming an
 * unsupplied card model keeps every value in the file, and a card naming an
 * unsupplied section is placed before the first section, so every supplied card
 * is drawn and counted.
 */
export function getGrid(deck: DeckGridSource, dragCard: string | null = null): Grid {
  const { cards, sections, cardModels } = deck
  const sectioned = new Set<string | null>(sections.map((section) => section.id))
  const placement: PlacementContext = { sectioned, dragCard }
  const tiles = cards.map((card) => createTile(card, cardModels, placement))

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

  // Every drawn element is numbered in drawing order: the tiles of each run, then
  // the run's plus, so each element's position among all is known.
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
 * Whether dropping the dragged card at the target moves it. A drop at the
 * card's own position moves nothing: the head of the deck is the position of the
 * first card under no section, and the end of a run is the position of its last
 * card.
 */
export const isMoved = (
  runs: readonly Run[],
  dragCard: string,
  target: InsertionPoint,
): boolean => {
  if (target === dragCard) return false
  if (target === HEAD) return runs[0]?.tiles[0]?.id !== dragCard

  const run = getRunEnd(target)
  return runs.find((each) => each.id === run)?.tiles.at(-1)?.id !== dragCard
}
