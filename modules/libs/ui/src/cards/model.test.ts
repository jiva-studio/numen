/**
 * What a stencil and a deck come to, as plain values. No DOM, no measurement:
 * every number and every flag a component draws is worked out here.
 */
import { describe, expect, it } from 'vitest'
import {
  aimedAt,
  blanks,
  faceBlocks,
  fieldRows,
  freeName,
  grid,
  laid,
  landing,
  objection,
  ordered,
  panes,
  parts,
  reordered,
  STENCIL_WORDS,
  type Cut,
  type Drawn,
  type Filled,
  type Shown,
} from './model'

describe('ordered', () => {
  const NAMES = ['a', 'b', 'c']

  it('puts a carried name before the one it lands on', () => {
    expect(ordered(NAMES, 'c', 'a')).toEqual(['c', 'a', 'b'])
  })

  it('puts it at the end where it lands on nothing', () => {
    expect(ordered(NAMES, 'a', null)).toEqual(['b', 'c', 'a'])
  })

  it('takes it out before it puts it back', () => {
    expect(ordered(NAMES, 'a', 'c')).toEqual(['b', 'a', 'c'])
  })

  it('leaves the order alone where it lands on itself', () => {
    expect(ordered(NAMES, 'b', 'b')).toEqual(NAMES)
  })

  it('leaves the order alone where it lands on a name that is not there', () => {
    expect(ordered(NAMES, 'b', 'z')).toEqual(NAMES)
  })

  it('leaves the order alone where what is carried is not there', () => {
    expect(ordered(NAMES, 'z', 'a')).toEqual(NAMES)
  })

  it('keeps one name alone where it is', () => {
    expect(ordered(['only'], 'only', null)).toEqual(['only'])
  })
})

describe('landing', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('lands a field below the first before another below the first', () => {
    expect(landing(FIELDS, 'Weight', 'Height')).toBe(true)
  })

  it('lands a field below the first at the end', () => {
    expect(landing(FIELDS, 'Height', null)).toBe(true)
  })

  it('does not move the first field, which names every card', () => {
    expect(landing(FIELDS, 'Name', 'Weight')).toBe(false)
    expect(landing(FIELDS, 'Name', null)).toBe(false)
  })

  it('lands nothing above the first field', () => {
    expect(landing(FIELDS, 'Weight', 'Name')).toBe(false)
  })

  it('moves nothing let go where it stands', () => {
    expect(landing(FIELDS, 'Height', 'Height')).toBe(false)
  })

  it('moves nothing among no fields', () => {
    expect(landing([], 'Height', null)).toBe(false)
  })

  it('does not move the one field a stencil names', () => {
    expect(landing(['Only'], 'Only', null)).toBe(false)
  })
})

describe('reordered', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('reorders the fields below the first', () => {
    expect(reordered(FIELDS, 'Weight', 'Height')).toEqual(['Name', 'Weight', 'Height'])
  })

  it('leaves the order alone where the first field is carried', () => {
    expect(reordered(FIELDS, 'Name', null)).toEqual(FIELDS)
  })

  it('leaves the order alone where a field is let go above the first', () => {
    expect(reordered(FIELDS, 'Weight', 'Name')).toEqual(FIELDS)
  })
})

describe('aimedAt', () => {
  it('aims at the front while nothing has been typed in', () => {
    expect(aimedAt(null, 'recognise')).toBe('front')
  })

  it('aims at the half last typed in', () => {
    expect(aimedAt({ face: 'recognise', half: 'back' }, 'recognise')).toBe('back')
  })

  it('aims at the front of a face other than the one last typed in', () => {
    expect(aimedAt({ face: 'recognise', half: 'back' }, 'name-it')).toBe('front')
  })
})

describe('objection', () => {
  it('objects to a name with nothing in it', () => {
    expect(objection('   ', [])).toBe('blank')
  })

  it('objects to a name already taken', () => {
    expect(objection('Height', ['Height'])).toBe('taken')
  })

  it('objects to a name taken but for the space around it', () => {
    expect(objection(' Height ', ['Height'])).toBe('taken')
  })

  it('objects to a name holding a brace, which no slot could write', () => {
    expect(objection('a{b', [])).toBe('braced')
    expect(objection('a}b', [])).toBe('braced')
  })

  it('takes a name that is free', () => {
    expect(objection('Weight', ['Height'])).toBeNull()
  })

  it('takes a name that is not Latin', () => {
    expect(objection('Продолжительность жизни', ['Height'])).toBeNull()
  })
})

describe('freeName', () => {
  it('numbers from one, so the first carries a number like the rest', () => {
    expect(freeName([], 'Field')).toBe('Field 1')
  })

  it('numbers past what is taken', () => {
    expect(freeName(['Field 1', 'Field 2'], 'Field')).toBe('Field 3')
  })

  it('fills a gap in the numbering', () => {
    expect(freeName(['Field 1', 'Field 3'], 'Field')).toBe('Field 2')
  })
})

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

  it('lays out nothing for a stencil naming nothing and a card holding nothing', () => {
    expect(laid([], [])).toEqual([])
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

describe('fieldRows', () => {
  const FIELDS = ['Height', 'Weight']

  it('numbers the rows from one, each knowing how many stand with it', () => {
    expect(fieldRows(FIELDS, null, null).map((row) => [row.at, row.of])).toEqual([
      [1, 2],
      [2, 2],
    ])
  })

  it('draws the name a field carries where nothing is being typed', () => {
    expect(fieldRows(FIELDS, null, null)[0]?.text).toBe('Height')
  })

  it('draws what is being typed over the name it is typed over', () => {
    const rows = fieldRows(FIELDS, { over: 'Height', text: 'Tall' }, null)
    expect(rows[0]?.text).toBe('Tall')
    expect(rows[1]?.text).toBe('Weight')
  })

  it('objects to nothing where nothing is being typed', () => {
    expect(fieldRows(FIELDS, null, null).every((row) => row.objection === null)).toBe(true)
  })

  it('objects where what is typed is another field’s name', () => {
    const rows = fieldRows(FIELDS, { over: 'Height', text: 'Weight' }, null)
    expect(rows[0]?.objection).toBe('taken')
  })

  it('objects to nothing where a field is typed its own name again', () => {
    const rows = fieldRows(FIELDS, { over: 'Height', text: 'Height' }, null)
    expect(rows[0]?.objection).toBeNull()
  })

  it('marks the one row on its way and no other', () => {
    const rows = fieldRows(FIELDS, null, 'Weight')
    expect(rows.map((row) => row.carried)).toEqual([false, true])
  })

  it('marks the first row as the one naming the cards, and no other', () => {
    expect(fieldRows(FIELDS, null, null).map((row) => row.names)).toEqual([true, false])
  })

  it('marks nothing as naming where a stencil names no field', () => {
    expect(fieldRows([], null, null)).toEqual([])
  })
})

describe('faceBlocks', () => {
  const FIELDS = ['Name', 'Height']
  const FACES: readonly Shown[] = [
    { id: 'recognise', name: 'Recognise', front: '{{Name}}', back: '{{Height}}' },
    { id: 'name-it', name: 'Name it', front: '{{Height}}', back: '{{Name}}' },
  ]
  const SAMPLE = [
    { field: 'Name', text: 'Llama' },
    { field: 'Height', text: 'about 45"' },
  ]

  it('numbers the blocks from one, each knowing how many stand with it', () => {
    expect(faceBlocks(FACES, FIELDS, SAMPLE).map((each) => [each.at, each.of])).toEqual([
      [1, 2],
      [2, 2],
    ])
  })

  it('shows each half with the sample values standing in it', () => {
    const block = faceBlocks(FACES, FIELDS, SAMPLE)[0]
    expect(block?.frontShown).toBe('Llama')
    expect(block?.backShown).toBe('about 45"')
  })

  it('keeps the markup a half was written with beside what it shows', () => {
    expect(faceBlocks(FACES, FIELDS, SAMPLE)[0]?.front).toBe('{{Name}}')
  })

  it('says the slots each half names that the fields do not, each once', () => {
    const faces: readonly Shown[] = [
      { id: 'one', name: 'One', front: '{{Colour}}', back: '{{Weight}} {{Weight}}' },
    ]
    const block = faceBlocks(faces, FIELDS, SAMPLE)[0]
    expect(block?.frontStray).toEqual(['Colour'])
    expect(block?.backStray).toEqual(['Weight'])
  })

  it('marks a stray slot in the preview where it stands, and fills the rest', () => {
    const faces: readonly Shown[] = [
      { id: 'one', name: 'One', front: '{{Name}} {{Colour}}', back: '' },
    ]
    expect(faceBlocks(faces, FIELDS, SAMPLE)[0]?.frontShown).toBe(
      'Llama <mark>{{Colour}}</mark>',
    )
  })

  it('says nothing stray of a face naming only declared fields', () => {
    const block = faceBlocks(FACES, FIELDS, SAMPLE)[0]
    expect(block?.frontStray).toEqual([])
    expect(block?.backStray).toEqual([])
  })

  it('draws no blocks for a stencil with no faces', () => {
    expect(faceBlocks([], FIELDS, SAMPLE)).toEqual([])
  })
})

describe('panes', () => {
  const FIELDS = ['Name', 'Height']
  const SAMPLE = [
    { field: 'Name', text: 'Llama' },
    { field: 'Height', text: 'about 45"' },
  ]

  const blockOf = (face: Shown, fields: readonly string[], sample: readonly Filled[]) => {
    const block = faceBlocks([face], fields, sample)[0]
    if (!block) throw new Error('no block')
    return block
  }

  const divided = (face: Shown) => panes(blockOf(face, FIELDS, SAMPLE))

  const FULL: Shown = {
    id: 'recognise',
    name: 'Recognise',
    front: '{{Name}}',
    back: '{{Height}}',
  }

  it('divides a face into four, the markup of each half before what it comes to', () => {
    expect(divided(FULL).map((pane) => [pane.half, pane.shows])).toEqual([
      ['front', 'written'],
      ['front', 'preview'],
      ['back', 'written'],
      ['back', 'preview'],
    ])
  })

  it('stands the markup in one part and the sample filling it in the next', () => {
    expect(divided(FULL).map((pane) => pane.text)).toEqual([
      '{{Name}}',
      'Llama',
      '{{Height}}',
      'about 45"',
    ])
  })

  it('calls each part what it holds', () => {
    expect(divided(FULL).map((pane) => pane.said)).toEqual([
      'Front',
      'Preview',
      'Back',
      'Preview',
    ])
  })

  it('announces a preview by the face and the half it is of', () => {
    expect(divided(FULL)[1]?.named).toBe('Preview: Recognise Front')
  })

  it('calls a part blank while nothing but space stands in it', () => {
    const face: Shown = { id: 'one', name: 'One', front: ' \n ', back: '{{Height}}' }
    expect(divided(face).map((pane) => pane.blank)).toEqual([true, true, false, false])
  })

  it('calls a part blank where what is written fills out to nothing', () => {
    const face: Shown = { id: 'one', name: 'One', front: '{{Blank}}', back: '' }
    const drawn = panes(blockOf(face, ['Blank'], [{ field: 'Blank', text: '' }]))
    expect(drawn[0]?.blank).toBe(false)
    expect(drawn[1]?.blank).toBe(true)
  })

  it('says a stray slot under the markup naming it, and not under the preview', () => {
    const face: Shown = { id: 'one', name: 'One', front: '{{Colour}}', back: '' }
    expect(divided(face).map((pane) => pane.stray)).toEqual([['Colour'], [], [], []])
  })

  it('draws every part with the words it was handed', () => {
    const drawn = panes(blockOf(FULL, FIELDS, SAMPLE), {
      ...STENCIL_WORDS,
      front: 'Recto',
      back: 'Verso',
      preview: 'As read',
    })
    expect(drawn.map((pane) => pane.said)).toEqual(['Recto', 'As read', 'Verso', 'As read'])
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
    })
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

  it('names no field where the card’s stencil was not handed in', () => {
    const orphan: readonly Drawn[] = [
      { id: 'x', name: 'X', stencil: 'Gone', filled: [{ field: 'A', text: 'a' }] },
    ]
    const tile = grid(orphan, CUTS, null).tiles[0]
    expect(tile?.known).toBe(false)
    expect(tile?.named).toBe(false)
    expect(tile?.filled).toEqual([
      { field: 'A', text: 'a', declared: false, names: false, twice: false, at: 1 },
    ])
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

describe('parts', () => {
  it('draws the front alone while the card is not turned', () => {
    expect(parts('f', 'b', false).map((part) => part.half)).toEqual(['front'])
  })

  it('draws the back under the front once the card is turned', () => {
    expect(parts('f', 'b', true).map((part) => part.half)).toEqual(['front', 'back'])
  })

  it('says a half holding nothing but space is blank', () => {
    expect(parts('  \n ', 'b', true).map((part) => part.blank)).toEqual([true, false])
  })

  it('hands the markdown on unchanged', () => {
    expect(parts('# Heading\n\n- one', '', false)[0]?.text).toBe('# Heading\n\n- one')
  })
})
