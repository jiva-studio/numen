/**
 * What a stencil evaluates to, as plain values. No DOM, no measurement: every
 * number and every flag a stencil draws is worked out here.
 */
import { describe, expect, it } from 'vitest'
import type { FieldValue } from './card'
import { faceRows, fieldRows, getPanes, STENCIL_WORDS, type StencilFace } from './stencil'

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
    expect(rows.map((row) => row.isDragged)).toEqual([false, true])
  })

  it('marks the first row as the one naming the cards, and no other', () => {
    expect(fieldRows(FIELDS, null).map((row) => row.hasNames)).toEqual([true, false])
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

describe('STENCIL_WORDS', () => {
  it('says what is wrong with a field name', () => {
    expect(STENCIL_WORDS.objection('blank')).toBe('A field needs a name')
    expect(STENCIL_WORDS.objection('taken')).toBe('That name is taken')
    expect(STENCIL_WORDS.objection('braced')).toBe('A name cannot hold a brace')
  })

  it('says what is wrong with a face name', () => {
    expect(STENCIL_WORDS.faceObjection('blank')).toBe('A face needs a name')
    expect(STENCIL_WORDS.faceObjection('taken')).toBe('That name is taken')
  })

  it('names every stray field, separated by commas', () => {
    expect(STENCIL_WORDS.stray(['A', 'B'])).toBe('Not a field: A, B')
  })
})

describe('faceRows', () => {
  const FIELDS = ['Name', 'Height']
  const FACES: readonly StencilFace[] = [
    { id: 'recognise', name: 'Recognise', front: '{{Name}}', back: '{{Height}}' },
    { id: 'name-it', name: 'Name it', front: '{{Height}}', back: '{{Name}}' },
  ]
  const SAMPLE = [
    { field: 'Name', text: 'Llama' },
    { field: 'Height', text: 'about 45"' },
  ]

  it('numbers the faces from one, each knowing how many stand with it', () => {
    expect(faceRows(FACES, FIELDS, SAMPLE).map((each) => [each.at, each.of])).toEqual([
      [1, 2],
      [2, 2],
    ])
  })

  it('shows each half with the sample values standing in it', () => {
    const face = faceRows(FACES, FIELDS, SAMPLE)[0]
    expect(face?.frontPreview).toBe('Llama')
    expect(face?.backPreview).toBe('about 45"')
  })

  it('keeps the markup a half was written with along with what it shows', () => {
    expect(faceRows(FACES, FIELDS, SAMPLE)[0]?.front).toBe('{{Name}}')
  })

  it('says the slots each half names that the fields do not, each once', () => {
    const faces: readonly StencilFace[] = [
      { id: 'one', name: 'One', front: '{{Colour}}', back: '{{Weight}} {{Weight}}' },
    ]
    const face = faceRows(faces, FIELDS, SAMPLE)[0]
    expect(face?.frontStray).toEqual(['Colour'])
    expect(face?.backStray).toEqual(['Weight'])
  })

  it('marks a stray slot in the preview at its current position, and fills the rest', () => {
    const faces: readonly StencilFace[] = [
      { id: 'one', name: 'One', front: '{{Name}} {{Colour}}', back: '' },
    ]
    expect(faceRows(faces, FIELDS, SAMPLE)[0]?.frontPreview).toBe('Llama <mark>{{Colour}}</mark>')
  })

  it('says nothing stray of a face naming only declared fields', () => {
    const face = faceRows(FACES, FIELDS, SAMPLE)[0]
    expect(face?.frontStray).toEqual([])
    expect(face?.backStray).toEqual([])
  })

  it('passes each face the names of the other faces', () => {
    expect(faceRows(FACES, FIELDS, SAMPLE).map((each) => each.taken)).toEqual([
      ['Name it'],
      ['Recognise'],
    ])
  })

  it('draws nothing for a stencil with no faces', () => {
    expect(faceRows([], FIELDS, SAMPLE)).toEqual([])
  })

  it('carries the identity and the name each face was handed in under', () => {
    const rendered = faceRows(FACES, FIELDS, SAMPLE)
    expect(rendered.map((each) => each.id)).toEqual(['recognise', 'name-it'])
    expect(rendered.map((each) => each.name)).toEqual(['Recognise', 'Name it'])
  })
})

describe('panes', () => {
  const FIELDS = ['Name', 'Height']
  const SAMPLE = [
    { field: 'Name', text: 'Llama' },
    { field: 'Height', text: 'about 45"' },
  ]

  const createFace = (
    face: StencilFace,
    fields: readonly string[],
    sample: readonly FieldValue[],
  ) => {
    const rendered = faceRows([face], fields, sample)[0]
    if (!rendered) throw new Error('no face')
    return rendered
  }

  const getFacePanes = (face: StencilFace) => getPanes(createFace(face, FIELDS, SAMPLE))

  const FULL: StencilFace = {
    id: 'recognise',
    name: 'Recognise',
    front: '{{Name}}',
    back: '{{Height}}',
  }

  it('divides a face into four, the markup of each half before what it evaluates to', () => {
    expect(getFacePanes(FULL).map((pane) => [pane.half, pane.mode])).toEqual([
      ['front', 'written'],
      ['front', 'preview'],
      ['back', 'written'],
      ['back', 'preview'],
    ])
  })

  it('remains the markup in one part and the sample filling it in the next', () => {
    expect(getFacePanes(FULL).map((pane) => pane.text)).toEqual([
      '{{Name}}',
      'Llama',
      '{{Height}}',
      'about 45"',
    ])
  })

  it('calls each part what it holds', () => {
    expect(getFacePanes(FULL).map((pane) => pane.said)).toEqual([
      'Front',
      'Preview',
      'Back',
      'Preview',
    ])
  })

  it('announces a preview by the face and the half it is of', () => {
    expect(getFacePanes(FULL)[1]?.named).toBe('Preview: Recognise Front')
  })

  it('calls a part blank while nothing but space remains in it', () => {
    const face: StencilFace = { id: 'one', name: 'One', front: ' \n ', back: '{{Height}}' }
    expect(getFacePanes(face).map((pane) => pane.isBlank)).toEqual([true, true, false, false])
  })

  it('calls a part blank where what is written fills out to nothing', () => {
    const face: StencilFace = { id: 'one', name: 'One', front: '{{Blank}}', back: '' }
    const rendered = getPanes(createFace(face, ['Blank'], [{ field: 'Blank', text: '' }]))
    expect(rendered[0]?.isBlank).toBe(false)
    expect(rendered[1]?.isBlank).toBe(true)
  })

  it('says a stray slot under the markup naming it, and not under the preview', () => {
    const face: StencilFace = { id: 'one', name: 'One', front: '{{Colour}}', back: '' }
    expect(getFacePanes(face).map((pane) => pane.stray)).toEqual([['Colour'], [], [], []])
  })

  it('draws every part with the words it was handed', () => {
    const rendered = getPanes(createFace(FULL, FIELDS, SAMPLE), {
      ...STENCIL_WORDS,
      front: 'Recto',
      back: 'Verso',
      preview: 'As read',
    })
    expect(rendered.map((pane) => pane.said)).toEqual(['Recto', 'As read', 'Verso', 'As read'])
  })
})
