/**
 * What a card model evaluates to, as plain values. No DOM, no measurement: every
 * number and every flag a card model draws is worked out here.
 */
import { describe, expect, it } from 'vitest'
import type { FieldValue } from './card'
import {
  templateRows,
  fieldRows,
  getPanes,
  CARD_MODEL_WORDS,
  type CardModelTemplate,
} from './card-model'

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

  it('marks nothing as naming where a card model names no field', () => {
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

describe('CARD_MODEL_WORDS', () => {
  it('says what is wrong with a field name', () => {
    expect(CARD_MODEL_WORDS.objection('blank')).toBe('A field needs a name')
    expect(CARD_MODEL_WORDS.objection('taken')).toBe('That name is taken')
    expect(CARD_MODEL_WORDS.objection('braced')).toBe('A name cannot hold a brace')
  })

  it('says what is wrong with a template name', () => {
    expect(CARD_MODEL_WORDS.templateObjection('blank')).toBe('A template needs a name')
    expect(CARD_MODEL_WORDS.templateObjection('taken')).toBe('That name is taken')
  })

  it('names every stray field, separated by commas', () => {
    expect(CARD_MODEL_WORDS.stray(['A', 'B'])).toBe('Not a field: A, B')
  })
})

describe('templateRows', () => {
  const FIELDS = ['Name', 'Height']
  const TEMPLATES: readonly CardModelTemplate[] = [
    { id: 'recognise', name: 'Recognise', front: '{{Name}}', back: '{{Height}}' },
    { id: 'name-it', name: 'Name it', front: '{{Height}}', back: '{{Name}}' },
  ]
  const SAMPLE = [
    { field: 'Name', text: 'Llama' },
    { field: 'Height', text: 'about 45"' },
  ]

  it('numbers the templates from one, each knowing how many stand with it', () => {
    expect(templateRows(TEMPLATES, FIELDS, SAMPLE).map((each) => [each.at, each.of])).toEqual([
      [1, 2],
      [2, 2],
    ])
  })

  it('shows each half with the sample values standing in it', () => {
    const template = templateRows(TEMPLATES, FIELDS, SAMPLE)[0]
    expect(template?.frontPreview).toBe('Llama')
    expect(template?.backPreview).toBe('about 45"')
  })

  it('keeps the markup a half was written with along with what it shows', () => {
    expect(templateRows(TEMPLATES, FIELDS, SAMPLE)[0]?.front).toBe('{{Name}}')
  })

  it('says the slots each half names that the fields do not, each once', () => {
    const templates: readonly CardModelTemplate[] = [
      { id: 'one', name: 'One', front: '{{Colour}}', back: '{{Weight}} {{Weight}}' },
    ]
    const template = templateRows(templates, FIELDS, SAMPLE)[0]
    expect(template?.frontStray).toEqual(['Colour'])
    expect(template?.backStray).toEqual(['Weight'])
  })

  it('marks a stray slot in the preview at its current position, and fills the rest', () => {
    const templates: readonly CardModelTemplate[] = [
      { id: 'one', name: 'One', front: '{{Name}} {{Colour}}', back: '' },
    ]
    expect(templateRows(templates, FIELDS, SAMPLE)[0]?.frontPreview).toBe(
      'Llama <mark>{{Colour}}</mark>',
    )
  })

  it('says nothing stray of a template naming only declared fields', () => {
    const template = templateRows(TEMPLATES, FIELDS, SAMPLE)[0]
    expect(template?.frontStray).toEqual([])
    expect(template?.backStray).toEqual([])
  })

  it('passes each template the names of the other templates', () => {
    expect(templateRows(TEMPLATES, FIELDS, SAMPLE).map((each) => each.taken)).toEqual([
      ['Name it'],
      ['Recognise'],
    ])
  })

  it('draws nothing for a card model with no templates', () => {
    expect(templateRows([], FIELDS, SAMPLE)).toEqual([])
  })

  it('carries the identity and the name each template was handed in under', () => {
    const renderedRows = templateRows(TEMPLATES, FIELDS, SAMPLE)
    expect(renderedRows.map((each) => each.id)).toEqual(['recognise', 'name-it'])
    expect(renderedRows.map((each) => each.name)).toEqual(['Recognise', 'Name it'])
  })
})

describe('panes', () => {
  const FIELDS = ['Name', 'Height']
  const SAMPLE = [
    { field: 'Name', text: 'Llama' },
    { field: 'Height', text: 'about 45"' },
  ]

  const createTemplate = (
    template: CardModelTemplate,
    fields: readonly string[],
    sample: readonly FieldValue[],
  ) => {
    const renderedRows = templateRows([template], fields, sample)[0]
    if (!renderedRows) throw new Error('no template')
    return renderedRows
  }

  const getTemplatePanes = (template: CardModelTemplate) =>
    getPanes(createTemplate(template, FIELDS, SAMPLE))

  const FULL: CardModelTemplate = {
    id: 'recognise',
    name: 'Recognise',
    front: '{{Name}}',
    back: '{{Height}}',
  }

  it('divides a template into four, the markup of each half before what it evaluates to', () => {
    expect(getTemplatePanes(FULL).map((pane) => [pane.half, pane.mode])).toEqual([
      ['front', 'written'],
      ['front', 'preview'],
      ['back', 'written'],
      ['back', 'preview'],
    ])
  })

  it('remains the markup in one part and the sample filling it in the next', () => {
    expect(getTemplatePanes(FULL).map((pane) => pane.text)).toEqual([
      '{{Name}}',
      'Llama',
      '{{Height}}',
      'about 45"',
    ])
  })

  it('calls each part what it holds', () => {
    expect(getTemplatePanes(FULL).map((pane) => pane.label)).toEqual([
      'Front',
      'Preview',
      'Back',
      'Preview',
    ])
  })

  it('announces a preview by the template and the half it is of', () => {
    expect(getTemplatePanes(FULL)[1]?.named).toBe('Preview: Recognise Front')
  })

  it('calls a part blank while nothing but space remains in it', () => {
    const template: CardModelTemplate = {
      id: 'one',
      name: 'One',
      front: ' \n ',
      back: '{{Height}}',
    }
    expect(getTemplatePanes(template).map((pane) => pane.isBlank)).toEqual([
      true,
      true,
      false,
      false,
    ])
  })

  it('calls a part blank where what is written fills out to nothing', () => {
    const template: CardModelTemplate = { id: 'one', name: 'One', front: '{{Blank}}', back: '' }
    const renderedRows = getPanes(
      createTemplate(template, ['Blank'], [{ field: 'Blank', text: '' }]),
    )
    expect(renderedRows[0]?.isBlank).toBe(false)
    expect(renderedRows[1]?.isBlank).toBe(true)
  })

  it('says a stray slot under the markup naming it, and not under the preview', () => {
    const template: CardModelTemplate = { id: 'one', name: 'One', front: '{{Colour}}', back: '' }
    expect(getTemplatePanes(template).map((pane) => pane.stray)).toEqual([['Colour'], [], [], []])
  })

  it('draws every part with the words it was handed', () => {
    const renderedRows = getPanes(createTemplate(FULL, FIELDS, SAMPLE), {
      ...CARD_MODEL_WORDS,
      front: 'Recto',
      back: 'Verso',
      preview: 'As read',
    })
    expect(renderedRows.map((pane) => pane.label)).toEqual(['Recto', 'As read', 'Verso', 'As read'])
  })
})
