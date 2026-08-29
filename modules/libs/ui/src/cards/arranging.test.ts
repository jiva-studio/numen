/**
 * What a stencil comes to, as plain values. No DOM, no measurement: every
 * number and every flag a stencil draws is worked out here.
 */
import { describe, expect, it } from 'vitest'
import type { Filled } from './deck'
import { faceBlocks, fieldRows, panes, STENCIL_WORDS, type Shown } from './stencil'

describe('fieldRows', () => {
  const FIELDS = ['Height', 'Weight']

  it('draws a row per field, in the order they were handed in', () => {
    expect(fieldRows(FIELDS, null).map((row) => row.field)).toEqual(FIELDS)
  })

  it('numbers the rows from one, each knowing how many stand with it', () => {
    expect(fieldRows(FIELDS, null).map((row) => [row.at, row.of])).toEqual([
      [1, 2],
      [2, 2],
    ])
  })

  it('marks the one row on its way and no other', () => {
    const rows = fieldRows(FIELDS, 'Weight')
    expect(rows.map((row) => row.carried)).toEqual([false, true])
  })

  it('marks the first row as the one naming the cards, and no other', () => {
    expect(fieldRows(FIELDS, null).map((row) => row.names)).toEqual([true, false])
  })

  it('marks nothing as naming where a stencil names no field', () => {
    expect(fieldRows([], null)).toEqual([])
  })

  it('draws one row for a name declared twice, and counts it once', () => {
    const rows = fieldRows(['Height', 'Weight', 'Height'], null)
    expect(rows.map((row) => row.field)).toEqual(['Height', 'Weight'])
    expect(rows.map((row) => [row.at, row.of])).toEqual([
      [1, 2],
      [2, 2],
    ])
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

  it('carries the identity and the name each face was handed in under', () => {
    const blocks = faceBlocks(FACES, FIELDS, SAMPLE)
    expect(blocks.map((each) => each.id)).toEqual(['recognise', 'name-it'])
    expect(blocks.map((each) => each.name)).toEqual(['Recognise', 'Name it'])
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
