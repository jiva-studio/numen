/**
 * What a deck comes to, as plain values. No DOM, no measurement: every number
 * and every flag a deck draws is worked out here.
 */
import { describe, expect, it } from 'vitest'
import { blanks, DECK_WORDS, grid, laid, NOTHING_WRONG, type Drawn } from './deck'
import type { Cut } from './stencil'

describe('laid', () => {
  const FIELDS = ['Height', 'Weight']

  it('lays the values in the order the stencil asks for them', () => {
    const filled = [
      { field: 'Weight', text: 'heavy' },
      { field: 'Height', text: 'tall' },
    ]
    expect(laid(filled, FIELDS).map((each) => each.field)).toEqual(FIELDS)
  })

  it('stands a field the card leaves out empty', () => {
    expect(laid([{ field: 'Height', text: 'tall' }], FIELDS)[1]).toEqual({
      field: 'Weight',
      text: '',
      declared: true,
    })
  })

  it('keeps a value the stencil does not name, after the rest', () => {
    const filled = [{ field: 'Colour', text: 'brown' }]
    expect(laid(filled, FIELDS)).toHaveLength(3)
    expect(laid(filled, FIELDS)[2]).toEqual({
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
    expect(laid(filled, FIELDS)).toEqual([
      { field: 'Height', text: 'tall', declared: true },
      { field: 'Height', text: 'taller', declared: true },
      { field: 'Weight', text: '', declared: true },
    ])
  })

  it('lays out nothing for a stencil naming nothing and a card holding nothing', () => {
    expect(laid([], [])).toEqual([])
  })

  it('lays a name the stencil declares twice out once, where it first stands', () => {
    const filled = [{ field: 'Height', text: 'tall' }]
    expect(laid(filled, ['Height', 'Weight', 'Height'])).toEqual([
      { field: 'Height', text: 'tall', declared: true },
      { field: 'Weight', text: '', declared: true },
    ])
  })
})

describe('blanks', () => {
  it('stands every field of a stencil empty', () => {
    expect(blanks(['A', 'B'])).toEqual([
      { field: 'A', text: '' },
      { field: 'B', text: '' },
    ])
  })
})

describe('grid', () => {
  const CUTS: readonly Cut[] = [
    { name: 'Animal', fields: ['Name', 'Height', 'Weight'] },
    { name: 'Word', fields: ['Word', 'Meaning'] },
  ]
  const CARDS: readonly Drawn[] = [
    // What names a card is the name it was handed, and it stands in none of
    // its values.
    {
      id: 'llama',
      name: 'Llama',
      stencil: 'Animal',
      filled: [{ field: 'Height', text: '45"' }],
    },
    { id: 'yak', name: '', stencil: 'Animal', filled: [] },
  ]

  it('stands the plus last, and counts it among the tiles', () => {
    const shown = grid(CARDS, CUTS, null)
    expect(shown.tiles.map((tile) => tile.at)).toEqual([1, 2])
    expect(shown.plusAt).toBe(3)
    expect(shown.of).toBe(3)
  })

  it('is one tile, the plus, where there are no cards', () => {
    const shown = grid([], CUTS, null)
    expect(shown.tiles).toEqual([])
    expect(shown.plusAt).toBe(1)
    expect(shown.of).toBe(1)
  })

  it('stands the field naming the card first among the values, and the rest after it', () => {
    const tile = grid(CARDS, CUTS, null).tiles[0]
    expect(tile?.filled.map((each) => each.field)).toEqual(['Name', 'Height', 'Weight'])
    expect(tile?.filled.map((each) => each.names)).toEqual([true, false, false])
    expect(tile?.known).toBe(true)
    expect(tile?.named).toBe(true)
  })

  it('stands the name the card was handed in the field that names it', () => {
    const tile = grid(CARDS, CUTS, null).tiles[0]
    expect(tile?.filled[0]).toEqual({
      field: 'Name',
      text: 'Llama',
      declared: true,
      names: true,
      twice: false,
      at: 1,
      nth: 0,
      key: 'Name#0',
      last: true,
    })
  })

  it('tells every tile how many stand in the grid, the plus among them', () => {
    expect(grid(CARDS, CUTS, null).tiles.map((tile) => tile.of)).toEqual([3, 3])
  })

  it('counts the values under a field from one, the name standing among none of them', () => {
    const said: readonly Drawn[] = [
      {
        id: 'x',
        name: 'Llama',
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const filled = grid(said, CUTS, null).tiles[0]?.filled ?? []
    expect(
      filled.filter((each) => each.field === 'Name').map((each) => [each.names, each.nth]),
    ).toEqual([
      [true, 0],
      [false, 1],
      [false, 2],
    ])
  })

  it('marks the last box standing for each field, which is where a mark is said', () => {
    const said: readonly Drawn[] = [
      { id: 'x', name: 'Llama', stencil: 'Animal', filled: [{ field: 'Name', text: 'Alpaca' }] },
    ]
    const filled = grid(said, CUTS, null).tiles[0]?.filled ?? []
    expect(filled.map((each) => [each.field, each.last])).toEqual([
      ['Name', false],
      ['Height', true],
      ['Weight', true],
      ['Name', true],
    ])
  })

  it('marks the one box standing for a field the card writes once', () => {
    const filled = grid(CARDS, CUTS, null).tiles[0]?.filled ?? []
    expect(filled.every((each) => each.last)).toBe(true)
  })

  it('draws a card cut by nothing as cut by nothing', () => {
    const bare: readonly Drawn[] = [{ id: 'x', name: 'X', stencil: null, filled: [] }]
    const tile = grid(bare, CUTS, null).tiles[0]
    expect(tile?.stencil).toBeNull()
    expect(tile?.known).toBe(false)
  })

  it('stands the naming field empty where the card was handed no name', () => {
    expect(grid(CARDS, CUTS, null).tiles[1]?.filled[0]?.text).toBe('')
  })

  it('numbers the values from one, so two of a name are still two values', () => {
    const said: readonly Drawn[] = [
      { id: 'x', name: 'Llama', stencil: 'Animal', filled: [{ field: 'Name', text: 'Alpaca' }] },
    ]
    const filled = grid(said, CUTS, null).tiles[0]?.filled ?? []
    expect(filled.map((each) => each.at)).toEqual([1, 2, 3, 4])
  })

  it('draws no value where the card’s stencil was not handed in', () => {
    const orphan: readonly Drawn[] = [
      { id: 'x', name: 'X', stencil: 'Gone', filled: [{ field: 'A', text: 'a' }] },
    ]
    const tile = grid(orphan, CUTS, null).tiles[0]
    expect(tile?.known).toBe(false)
    expect(tile?.named).toBe(false)
    // Nothing names these values, so nothing lays them out. They stay in the
    // file, and the tile says which stencil it is waiting for.
    expect(tile?.filled).toEqual([])
    expect(tile?.stencil).toBe('Gone')
  })

  it('names a card by its heading, and by no value it carries', () => {
    const said: readonly Drawn[] = [
      { id: 'x', name: 'Llama', stencil: 'Animal', filled: [{ field: 'Name', text: 'Alpaca' }] },
    ]
    expect(grid(said, CUTS, null).tiles[0]?.filled[0]?.text).toBe('Llama')
  })

  it('keeps a value standing in the field that names the card, and marks it', () => {
    const said: readonly Drawn[] = [
      { id: 'x', name: 'Llama', stencil: 'Animal', filled: [{ field: 'Name', text: 'Alpaca' }] },
    ]
    const filled = grid(said, CUTS, null).tiles[0]?.filled ?? []
    expect(filled.filter((each) => each.twice).map((each) => each.text)).toEqual(['Alpaca'])
    // It stands after the fields the stencil asks for, and takes none of their places.
    expect(filled.map((each) => each.field)).toEqual(['Name', 'Height', 'Weight', 'Name'])
  })

  it('marks no value as named twice where the card leaves that field out', () => {
    const filled = grid(CARDS, CUTS, null).tiles[0]?.filled
    expect(filled?.every((each) => !each.twice)).toBe(true)
  })

  it('says nothing of what a card is cut by: the fields tell the stencils apart', () => {
    const mixed: readonly Drawn[] = [
      ...CARDS,
      { id: 'llano', name: 'llano', stencil: 'Word', filled: [] },
    ]
    const shown = grid(mixed, CUTS, null)
    expect(shown.tiles.map((tile) => tile.stencil)).toEqual(['Animal', 'Animal', 'Word'])
    expect(Object.keys(shown.tiles[0] ?? {})).not.toContain('cut')
  })

  it('marks the one tile on its way and no other', () => {
    expect(grid(CARDS, CUTS, 'llama').tiles.map((tile) => tile.carried)).toEqual([true, false])
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
