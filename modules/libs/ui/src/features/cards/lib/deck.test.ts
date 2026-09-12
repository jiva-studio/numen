/** What a deck comes to, as plain values. No DOM, no measurement. */
import { describe, expect, it } from 'vitest'
import {
  getBlanks,
  DECK_WORDS,
  getCardFieldValues,
  NOTHING_WRONG,
  type DeckSection,
  type DeckCard,
} from './deck'
import { getGrid } from './grid'
import type { Stencil } from './card'

describe('getCardFieldValues', () => {
  const FIELDS = ['Height', 'Weight']

  it('lays the values in the order the stencil asks for them', () => {
    const filled = [
      { field: 'Weight', text: 'heavy' },
      { field: 'Height', text: 'tall' },
    ]
    expect(getCardFieldValues(filled, FIELDS).map((each) => each.field)).toEqual(FIELDS)
  })

  it('stands a field the card leaves out empty', () => {
    expect(getCardFieldValues([{ field: 'Height', text: 'tall' }], FIELDS)[1]).toEqual({
      field: 'Weight',
      text: '',
      declared: true,
    })
  })

  it('keeps a value the stencil does not name, after the rest', () => {
    const filled = [{ field: 'Colour', text: 'brown' }]
    expect(getCardFieldValues(filled, FIELDS)).toHaveLength(3)
    expect(getCardFieldValues(filled, FIELDS)[2]).toEqual({
      field: 'Colour',
      text: 'brown',
      declared: false,
    })
  })

  it('stands every value of a field the card writes more than once', () => {
    const filled = [
      { field: 'Height', text: 'tall' },
      { field: 'Height', text: 'taller' },
    ]
    expect(getCardFieldValues(filled, FIELDS)).toEqual([
      { field: 'Height', text: 'tall', declared: true },
      { field: 'Height', text: 'taller', declared: true },
      { field: 'Weight', text: '', declared: true },
    ])
  })

  it('lays out nothing for a stencil naming nothing and a card holding nothing', () => {
    expect(getCardFieldValues([], [])).toEqual([])
  })

  it('lays a name the stencil declares twice out once, where it first stands', () => {
    const filled = [{ field: 'Height', text: 'tall' }]
    expect(getCardFieldValues(filled, ['Height', 'Weight', 'Height'])).toEqual([
      { field: 'Height', text: 'tall', declared: true },
      { field: 'Weight', text: '', declared: true },
    ])
  })
})

describe('getBlanks', () => {
  it('stands every field of a stencil empty', () => {
    expect(getBlanks(['A', 'B'])).toEqual([
      { field: 'A', text: '' },
      { field: 'B', text: '' },
    ])
  })
})

describe('grid', () => {
  const CUTS: readonly Stencil[] = [
    { name: 'Animal', fields: ['Name', 'Height', 'Weight'] },
    { name: 'Word', fields: ['Word', 'Meaning'] },
  ]
  const CARDS: readonly DeckCard[] = [
    {
      id: 'llama',
      section: null,
      stencil: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: '45"' },
      ],
    },
    { id: 'yak', section: null, stencil: 'Animal', filled: [] },
  ]

  /** Every tile of the grid, over all its runs, in the order they stand. */
  const tilesOf = (deck: ReturnType<typeof getGrid>) => deck.runs.flatMap((run) => run.tiles)

  it('stands the plus last, and counts it among the tiles', () => {
    const shown = getGrid(CARDS, [], CUTS, null)
    expect(tilesOf(shown).map((tile) => tile.at)).toEqual([1, 2])
    expect(shown.runs[0]?.plusAt).toBe(3)
    expect(shown.of).toBe(3)
  })

  it('is one tile, the plus, where there are no cards', () => {
    const shown = getGrid([], [], CUTS, null)
    expect(tilesOf(shown)).toEqual([])
    expect(shown.runs[0]?.plusAt).toBe(1)
    expect(shown.of).toBe(1)
  })

  it('lays every field the stencil asks for out under its own name, the first included', () => {
    const tile = tilesOf(getGrid(CARDS, [], CUTS, null))[0]
    expect(tile?.filled.map((each) => each.field)).toEqual(['Name', 'Height', 'Weight'])
    expect(tile?.known).toBe(true)
  })

  it('stands the first field where the card wrote it, as it stands every other', () => {
    const tile = tilesOf(getGrid(CARDS, [], CUTS, null))[0]
    expect(tile?.filled[0]).toEqual({
      field: 'Name',
      text: 'Llama',
      declared: true,
      at: 1,
      nth: 1,
      key: 'Name#1',
      last: true,
    })
  })

  it('tells every tile how many stand in the grid, the plus among them', () => {
    expect(tilesOf(getGrid(CARDS, [], CUTS, null)).map((tile) => tile.of)).toEqual([3, 3])
  })

  it('counts the values under a field from one, the first field among them', () => {
    const said: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = tilesOf(getGrid(said, [], CUTS, null))[0]?.filled ?? []
    expect(filled.filter((each) => each.field === 'Name').map((each) => each.nth)).toEqual([1, 2])
  })

  it('marks the last box standing for each field, which is where a mark is said', () => {
    const said: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = tilesOf(getGrid(said, [], CUTS, null))[0]?.filled ?? []
    expect(filled.map((each) => [each.field, each.last])).toEqual([
      ['Name', false],
      ['Name', true],
      ['Height', true],
      ['Weight', true],
    ])
  })

  it('marks the one box standing for a field the card writes once', () => {
    const filled = tilesOf(getGrid(CARDS, [], CUTS, null))[0]?.filled ?? []
    expect(filled.every((each) => each.last)).toBe(true)
  })

  it('draws a card cut by nothing as cut by nothing', () => {
    const bare: readonly DeckCard[] = [{ id: 'x', section: null, stencil: null, filled: [] }]
    const tile = tilesOf(getGrid(bare, [], CUTS, null))[0]
    expect(tile?.stencil).toBeNull()
    expect(tile?.known).toBe(false)
  })

  it('stands every value of a card cut by nothing, marked as named by nothing', () => {
    const bare: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        stencil: null,
        filled: [
          { field: 'Question', text: 'what' },
          { field: 'Answer', text: 'this' },
        ],
      },
    ]
    const filled = tilesOf(getGrid(bare, [], CUTS, null))[0]?.filled ?? []
    expect(filled.map((each) => each.field)).toEqual(['Question', 'Answer'])
    expect(filled.map((each) => each.text)).toEqual(['what', 'this'])
    expect(filled.every((each) => !each.declared)).toBe(true)
  })

  it('stands a field the card leaves out empty', () => {
    expect(tilesOf(getGrid(CARDS, [], CUTS, null))[1]?.filled[0]?.text).toBe('')
  })

  it('numbers the values from one, so two of a name are still two values', () => {
    const said: readonly DeckCard[] = [
      {
        id: 'x',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = tilesOf(getGrid(said, [], CUTS, null))[0]?.filled ?? []
    expect(filled.map((each) => each.at)).toEqual([1, 2, 3, 4])
  })

  it('draws no value where the card’s stencil was not handed in', () => {
    const orphan: readonly DeckCard[] = [
      { id: 'x', section: null, stencil: 'Gone', filled: [{ field: 'A', text: 'a' }] },
    ]
    const tile = tilesOf(getGrid(orphan, [], CUTS, null))[0]
    expect(tile?.known).toBe(false)
    // Nothing names these values, so nothing lays them out. They stay in the
    // file, and the tile says which stencil it is waiting for.
    expect(tile?.filled).toEqual([])
    expect(tile?.stencil).toBe('Gone')
  })

  it('says nothing of what a card is cut by: the fields tell the stencils apart', () => {
    const mixed: readonly DeckCard[] = [
      ...CARDS,
      { id: 'llano', section: null, stencil: 'Word', filled: [] },
    ]
    const shown = getGrid(mixed, [], CUTS, null)
    expect(tilesOf(shown).map((tile) => tile.stencil)).toEqual(['Animal', 'Animal', 'Word'])
    expect(Object.keys(tilesOf(shown)[0] ?? {})).not.toContain('cut')
  })

  it('marks the one tile on its way and no other', () => {
    expect(tilesOf(getGrid(CARDS, [], CUTS, 'llama')).map((tile) => tile.dragged)).toEqual([
      true,
      false,
    ])
  })

  describe('the sections of a deck', () => {
    const SECTIONS: readonly DeckSection[] = [
      { id: 'roots', name: 'Roots' },
      { id: 'leaves', name: 'Leaves' },
    ]
    const SECTIONED: readonly DeckCard[] = [
      { id: 'loose', section: null, stencil: 'Animal', filled: [] },
      { id: 'llama', section: 'roots', stencil: 'Animal', filled: [] },
      { id: 'yak', section: 'roots', stencil: 'Animal', filled: [] },
    ]

    it('stands the cards before the first section in a run under no section', () => {
      const runs = getGrid(SECTIONED, SECTIONS, CUTS, null).runs
      expect(runs[0]?.section).toBeNull()
      expect(runs[0]?.tiles.map((tile) => tile.id)).toEqual(['loose'])
    })

    it('stands one run under each section, in the order the sections were handed in', () => {
      const runs = getGrid(SECTIONED, SECTIONS, CUTS, null).runs
      expect(runs.map((run) => run.section?.name ?? null)).toEqual([null, 'Roots', 'Leaves'])
      expect(runs.map((run) => run.section?.at ?? null)).toEqual([null, 1, 2])
    })

    it('keeps a section no card stands under, and draws it holding none', () => {
      const runs = getGrid(SECTIONED, SECTIONS, CUTS, null).runs
      expect(runs[2]?.section?.id).toBe('leaves')
      expect(runs[2]?.tiles).toEqual([])
    })

    it('says of each tile which section it stands under', () => {
      const runs = getGrid(SECTIONED, SECTIONS, CUTS, null).runs
      expect(runs[1]?.tiles.map((tile) => [tile.id, tile.section])).toEqual([
        ['llama', 'roots'],
        ['yak', 'roots'],
      ])
    })

    it('counts a tile’s place over the whole deck, and not over its run', () => {
      const runs = getGrid(SECTIONED, SECTIONS, CUTS, null).runs
      expect(runs.flatMap((run) => run.tiles).map((tile) => tile.at)).toEqual([1, 3, 4])
    })

    // A card is made at the end of a run, so every run cards may be put in
    // carries a plus, and each stands where it is drawn among them all.
    it('counts each plus where it stands, and every tile against them all', () => {
      const shown = getGrid(SECTIONED, SECTIONS, CUTS, null)
      expect(shown.runs.map((run) => run.plusAt)).toEqual([2, 5, 6])
      expect(shown.of).toBe(6)
      expect(shown.runs.flatMap((run) => run.tiles).map((tile) => tile.of)).toEqual([6, 6, 6])
    })

    it('stands no plus before the first section where no card stands there', () => {
      const under = SECTIONED.filter((card) => card.section !== null)
      expect(getGrid(under, SECTIONS, CUTS, null).runs.map((run) => run.plusAt)).toEqual([null, 3, 4])
    })

    it('stands one run, holding every card, where the deck has no section', () => {
      const runs = getGrid(CARDS, [], CUTS, null).runs
      expect(runs).toHaveLength(1)
      expect(runs[0]?.section).toBeNull()
    })
  })
})

describe('DECK_WORDS', () => {
  it('names the stencil a card is waiting for', () => {
    expect(DECK_WORDS.unknown('Gone')).toBe('No stencil called Gone')
  })

  it('says of a card cut by nothing that nothing cut it', () => {
    expect(DECK_WORDS.unknown(null)).toBe('Cut by no stencil')
  })
})

describe('NOTHING_WRONG', () => {
  it('holds nothing against any card and nothing against any value', () => {
    expect(NOTHING_WRONG.at.size).toBe(0)
    expect(NOTHING_WRONG.under.size).toBe(0)
  })

  it('stands for every caller at once, so nothing can be put into it', () => {
    const at = NOTHING_WRONG.at as Map<string, readonly string[]>
    expect(() => at.set('llama', ['put here by one caller'])).toThrow()
    expect(NOTHING_WRONG.at.size).toBe(0)

    const under = NOTHING_WRONG.under as Map<string, ReadonlyMap<string, readonly string[]>>
    expect(() => under.set('llama', new Map())).toThrow()
    expect(NOTHING_WRONG.under.size).toBe(0)
  })
})
