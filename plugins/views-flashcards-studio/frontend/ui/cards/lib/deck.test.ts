/** What a deck evaluates to, as plain values. No DOM, no measurement. */
import { describe, expect, it } from 'vitest'
import {
  getBlanks,
  DECK_WORDS,
  getCardFieldValues,
  NO_DECK_PROBLEMS,
  type DeckSection,
  type DeckCard,
} from './deck'
import { getGrid } from './grid'
import type { CardModel } from './card'

describe('getCardFieldValues', () => {
  const FIELDS = ['Height', 'Weight']

  it('lays the values in the order the card model asks for them', () => {
    const filled = [
      { field: 'Weight', text: 'heavy' },
      { field: 'Height', text: 'tall' },
    ]
    expect(getCardFieldValues(filled, FIELDS).map((each) => each.field)).toEqual(FIELDS)
  })

  it('remains a field the card leaves out empty', () => {
    expect(getCardFieldValues([{ field: 'Height', text: 'tall' }], FIELDS)[1]).toEqual({
      field: 'Weight',
      text: '',
      isDeclared: true,
    })
  })

  it('keeps a value the card model does not name, after the rest', () => {
    const filled = [{ field: 'Colour', text: 'brown' }]
    expect(getCardFieldValues(filled, FIELDS)).toHaveLength(3)
    expect(getCardFieldValues(filled, FIELDS)[2]).toEqual({
      field: 'Colour',
      text: 'brown',
      isDeclared: false,
    })
  })

  it('remains every value of a field the card writes more than once', () => {
    const filled = [
      { field: 'Height', text: 'tall' },
      { field: 'Height', text: 'taller' },
    ]
    expect(getCardFieldValues(filled, FIELDS)).toEqual([
      { field: 'Height', text: 'tall', isDeclared: true },
      { field: 'Height', text: 'taller', isDeclared: true },
      { field: 'Weight', text: '', isDeclared: true },
    ])
  })

  it('lays out nothing for a card model naming nothing and a card holding nothing', () => {
    expect(getCardFieldValues([], [])).toEqual([])
  })

  it('lays a name the card model declares twice out once, where it first remains', () => {
    const filled = [{ field: 'Height', text: 'tall' }]
    expect(getCardFieldValues(filled, ['Height', 'Weight', 'Height'])).toEqual([
      { field: 'Height', text: 'tall', isDeclared: true },
      { field: 'Weight', text: '', isDeclared: true },
    ])
  })
})

describe('getBlanks', () => {
  it('remains every field of a card model empty', () => {
    expect(getBlanks(['A', 'B'])).toEqual([
      { field: 'A', text: '' },
      { field: 'B', text: '' },
    ])
  })
})

describe('grid', () => {
  const CUTS: readonly CardModel[] = [
    { name: 'Animal', fields: ['Name', 'Height', 'Weight'] },
    { name: 'Word', fields: ['Word', 'Meaning'] },
  ]
  const CARDS: readonly DeckCard[] = [
    {
      id: 'llama',
      section: null,
      cardModel: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: '45"' },
      ],
    },
    { id: 'yak', section: null, cardModel: 'Animal', filled: [] },
  ]

  const grid = (
    cards: readonly DeckCard[],
    sections: readonly DeckSection[] = [],
    dragCard: string | null = null,
  ) => getGrid({ cards, sections, cardModels: CUTS }, dragCard)

  /** Every tile of the grid, over all its runs, in the order they stand. */
  const getTiles = (deck: ReturnType<typeof getGrid>) => deck.runs.flatMap((run) => run.tiles)

  it('remains the plus last, and counts it among the tiles', () => {
    const displayedGrid = grid(CARDS)
    expect(getTiles(displayedGrid).map((tile) => tile.at)).toEqual([1, 2])
    expect(displayedGrid.runs[0]?.plusAt).toBe(3)
    expect(displayedGrid.of).toBe(3)
  })

  it('is one tile, the plus, where there are no cards', () => {
    const displayedGrid = grid([])
    expect(getTiles(displayedGrid)).toEqual([])
    expect(displayedGrid.runs[0]?.plusAt).toBe(1)
    expect(displayedGrid.of).toBe(1)
  })

  it('lays every field the card model asks for out under its own name, the first included', () => {
    const tile = getTiles(grid(CARDS))[0]
    expect(tile?.filled.map((each) => each.field)).toEqual(['Name', 'Height', 'Weight'])
    expect(tile?.isKnown).toBe(true)
  })

  it('remains the first field where the card wrote it, currently every other', () => {
    const tile = getTiles(grid(CARDS))[0]
    expect(tile?.filled[0]).toEqual({
      field: 'Name',
      text: 'Llama',
      isDeclared: true,
      at: 1,
      index: 1,
      key: 'Name#1',
      isLast: true,
    })
  })

  it('notifies every tile how many stand in the grid, the plus among them', () => {
    expect(getTiles(grid(CARDS)).map((tile) => tile.of)).toEqual([3, 3])
  })

  it('counts the values under a field from one, the first field among them', () => {
    const response: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = getTiles(grid(response))[0]?.filled ?? []
    expect(filled.filter((each) => each.field === 'Name').map((each) => each.index)).toEqual([1, 2])
  })

  it('marks the last box standing for each field, which is where a mark is returned', () => {
    const response: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = getTiles(grid(response))[0]?.filled ?? []
    expect(filled.map((each) => [each.field, each.isLast])).toEqual([
      ['Name', false],
      ['Name', true],
      ['Height', true],
      ['Weight', true],
    ])
  })

  it('marks the one box standing for a field the card writes once', () => {
    const filled = getTiles(grid(CARDS))[0]?.filled ?? []
    expect(filled.every((each) => each.isLast)).toBe(true)
  })

  it('draws a card cut by nothing as cut by nothing', () => {
    const withoutCardModelCards: readonly DeckCard[] = [
      { id: 'x', section: null, cardModel: null, filled: [] },
    ]
    const tile = getTiles(grid(withoutCardModelCards))[0]
    expect(tile?.cardModel).toBeNull()
    expect(tile?.isKnown).toBe(false)
  })

  it('remains every value of a card cut by nothing, marked as named by nothing', () => {
    const withoutCardModelCards: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        cardModel: null,
        filled: [
          { field: 'Question', text: 'what' },
          { field: 'Answer', text: 'this' },
        ],
      },
    ]
    const filled = getTiles(grid(withoutCardModelCards))[0]?.filled ?? []
    expect(filled.map((each) => each.field)).toEqual(['Question', 'Answer'])
    expect(filled.map((each) => each.text)).toEqual(['what', 'this'])
    expect(filled.every((each) => !each.isDeclared)).toBe(true)
  })

  it('remains a field the card leaves out empty', () => {
    expect(getTiles(grid(CARDS))[1]?.filled[0]?.text).toBe('')
  })

  it('numbers the values from one, so two of a name are still two values', () => {
    const response: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = getTiles(grid(response))[0]?.filled ?? []
    expect(filled.map((each) => each.at)).toEqual([1, 2, 3, 4])
  })

  it('draws no value where the card’s card model was not handed in', () => {
    const orphan: readonly DeckCard[] = [
      { id: 'x', section: null, cardModel: 'Gone', filled: [{ field: 'A', text: 'a' }] },
    ]
    const tile = getTiles(grid(orphan))[0]
    expect(tile?.isKnown).toBe(false)
    // No card model matches these values, so no template renders them. They stay
    // in the file, and the tile shows which card model it is waiting for.
    expect(tile?.filled).toEqual([])
    expect(tile?.cardModel).toBe('Gone')
  })

  it('says nothing of what a card is cut by: the fields tell the card models apart', () => {
    const mixed: readonly DeckCard[] = [
      ...CARDS,
      { id: 'llano', section: null, cardModel: 'Word', filled: [] },
    ]
    const displayedGrid = grid(mixed)
    expect(getTiles(displayedGrid).map((tile) => tile.cardModel)).toEqual([
      'Animal',
      'Animal',
      'Word',
    ])
    expect(Object.keys(getTiles(displayedGrid)[0] ?? {})).not.toContain('cut')
  })

  it('marks the one tile on its way and no other', () => {
    expect(getTiles(grid(CARDS, [], 'llama')).map((tile) => tile.isDragged)).toEqual([true, false])
  })

  describe('the sections of a deck', () => {
    const SECTIONS: readonly DeckSection[] = [
      { id: 'roots', name: 'Roots' },
      { id: 'leaves', name: 'Leaves' },
    ]
    const SECTIONED: readonly DeckCard[] = [
      { id: 'loose', section: null, cardModel: 'Animal', filled: [] },
      { id: 'llama', section: 'roots', cardModel: 'Animal', filled: [] },
      { id: 'yak', section: 'roots', cardModel: 'Animal', filled: [] },
    ]

    it('remains the cards before the first section in a run under no section', () => {
      const runs = grid(SECTIONED, SECTIONS).runs
      expect(runs[0]?.section).toBeNull()
      expect(runs[0]?.tiles.map((tile) => tile.id)).toEqual(['loose'])
    })

    it('remains one run under each section, in the order the sections were handed in', () => {
      const runs = grid(SECTIONED, SECTIONS).runs
      expect(runs.map((run) => run.section?.name ?? null)).toEqual([null, 'Roots', 'Leaves'])
      expect(runs.map((run) => run.section?.at ?? null)).toEqual([null, 1, 2])
    })

    it('keeps a section no card is placed under, and draws it holding none', () => {
      const runs = grid(SECTIONED, SECTIONS).runs
      expect(runs[2]?.section?.id).toBe('leaves')
      expect(runs[2]?.tiles).toEqual([])
    })

    it('says of each tile which section it is placed under', () => {
      const runs = grid(SECTIONED, SECTIONS).runs
      expect(runs[1]?.tiles.map((tile) => [tile.id, tile.section])).toEqual([
        ['llama', 'roots'],
        ['yak', 'roots'],
      ])
    })

    it('counts a tile’s place over the whole deck, and not over its run', () => {
      const runs = grid(SECTIONED, SECTIONS).runs
      expect(runs.flatMap((run) => run.tiles).map((tile) => tile.at)).toEqual([1, 3, 4])
    })

    // A card is added at the end of a run, so every run that cards may be added
    // to has a plus, and each plus has a position among all of them.
    it('counts each plus at its current position, and every tile against them all', () => {
      const displayedGrid = grid(SECTIONED, SECTIONS)
      expect(displayedGrid.runs.map((run) => run.plusAt)).toEqual([2, 5, 6])
      expect(displayedGrid.of).toBe(6)
      expect(displayedGrid.runs.flatMap((run) => run.tiles).map((tile) => tile.of)).toEqual([
        6, 6, 6,
      ])
    })

    it('remains no plus before the first section where no card remains there', () => {
      const underItem = SECTIONED.filter((card) => card.section !== null)
      expect(grid(underItem, SECTIONS).runs.map((run) => run.plusAt)).toEqual([null, 3, 4])
    })

    it('remains one run, holding every card, where the deck has no section', () => {
      const runs = grid(CARDS).runs
      expect(runs).toHaveLength(1)
      expect(runs[0]?.section).toBeNull()
    })
  })
})

describe('DECK_WORDS', () => {
  it('names the card model a card is waiting for', () => {
    expect(DECK_WORDS.unknown('Gone')).toBe('No card model called Gone')
  })

  it('says of a card cut by nothing that nothing cut it', () => {
    expect(DECK_WORDS.unknown(null)).toBe('Cut by no card model')
  })
})

describe('NO_DECK_PROBLEMS', () => {
  it('holds nothing against any card and nothing against any value', () => {
    expect(NO_DECK_PROBLEMS.at.size).toBe(0)
    expect(NO_DECK_PROBLEMS.under.size).toBe(0)
  })

  it('remains for every caller at once, so nothing can be put into it', () => {
    const problems = NO_DECK_PROBLEMS.at as Map<string, readonly string[]>
    expect(() => problems.set('llama', ['put here by one caller'])).toThrow()
    expect(NO_DECK_PROBLEMS.at.size).toBe(0)

    const underItem = NO_DECK_PROBLEMS.under as Map<string, ReadonlyMap<string, readonly string[]>>
    expect(() => underItem.set('llama', new Map())).toThrow()
    expect(NO_DECK_PROBLEMS.under.size).toBe(0)
  })
})
