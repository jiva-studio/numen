import { describe, expect, it } from 'vitest'
import {
  countCards,
  createCardId,
  createCardModelText,
  createCardText,
  createDeckText,
  createPresetText,
  createTemplatesText,
  getCardFile,
  parseFieldNames,
  parsePresetSettings,
  parseTemplates,
  parseValues,
  readCardModelFields,
} from './cards.js'

describe('getCardFile', () => {
  it('puts a card file in the cards folder, named by its title', () => {
    expect(getCardFile('Animal')).toBe('cards/Animal.md')
  })

  it('refuses a title that is a path', () => {
    expect(() => getCardFile('a/b')).toThrow('cannot name a file')
  })
})

describe('parseFieldNames', () => {
  it('reads one row of names', () => {
    expect(parseFieldNames([[' Name ', 'Height']])).toEqual(['Name', 'Height'])
  })

  it.each([[[]], [[['A'], ['B']]], [[['A', '']]], [[['A', 'A']]]])('refuses %j', (rows) => {
    expect(() => parseFieldNames(rows)).toThrow()
  })
})

describe('parseTemplates and parseValues', () => {
  it('reads the rows under the header', () => {
    expect(parseTemplates([['Template', 'Front', 'Back'], ['Recognise', '{{Name}}', 'Tall']])).toEqual([
      { name: 'Recognise', front: '{{Name}}', back: 'Tall' },
    ])
    expect(parseValues([['Field', 'Value'], ['Name', 'Llama']])).toEqual([['Name', 'Llama']])
  })

  it('names the header it expects', () => {
    expect(() => parseTemplates([['Name', 'Front', 'Back'], ['a', 'b', 'c']])).toThrow(
      'the templates is a table with the header "Template", "Front", "Back"',
    )
  })

  it('refuses a table without rows', () => {
    expect(() => parseValues([['Field', 'Value']])).toThrow('has no row')
  })
})

describe('the text of a card model', () => {
  const text = createCardModelText('Animal', ['Name', 'Ареал обитания']) + createTemplatesText([
    { name: 'Recognise', front: '{{Name}}', back: '**Range:** {{Ареал обитания}}' },
  ])

  it('lists the fields in its frontmatter and the templates as sections', () => {
    expect(text.startsWith('---\ntype: card-model\ntitle: Animal\nfields:\n  - Name\n')).toBe(true)
    expect(text).toContain('## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n**Range:** {{Ареал обитания}}\n')
  })

  it('gives its fields back', () => {
    expect(readCardModelFields(text)).toEqual(['Name', 'Ареал обитания'])
  })

  it('reads the fields of the card model of the vault of the product', () => {
    expect(readCardModelFields('---\ntype: card-model\ntitle: Animal\nfields:\n  - Name\n  - Life span\ncustom: x\n---\n')).toEqual([
      'Name',
      'Life span',
    ])
  })
})

describe('the text of a deck', () => {
  const id = createCardId('Animals', 0)

  it('makes ids of ten letters the product accepts, one for each card', () => {
    expect(id).toMatch(/^[0-9a-hjkmnp-tv-z]{10}$/)
    expect(createCardId('Animals', 1)).not.toBe(id)
    expect(createCardId('Animals', 0)).toBe(id)
  })

  it('has a card as a section with the name of its card model', () => {
    const card = createCardText('Animal', ['Name', 'Height'], [['Height', 'tall'], ['Name', 'Llama']], id)
    expect(card).toBe(`\n## Llama ^${id}\n\n[[Animal]]\n\n### Height\n\ntall\n\n### Name\n\nLlama\n`)
    expect(countCards(createDeckText('Animals') + card + card)).toBe(2)
  })

  it('refuses a value of a field the card model does not have', () => {
    expect(() => createCardText('Animal', ['Name'], [['Weight', '1']], id)).toThrow(
      'the card model "Animal" has no field "Weight": it has Name',
    )
  })

  it('starts without cards', () => {
    expect(createDeckText('Animals')).toBe('---\ntype: deck\ntitle: Animals\n---\n')
    expect(countCards(createDeckText('Animals'))).toBe(0)
  })
})

describe('the settings of a preset', () => {
  it('turns the words of the preset tab into the keys of the file', () => {
    const settings = parsePresetSettings([
      ['Setting', 'Value'],
      ['Goal', 'Minutes per day'],
      ['Minutes per day', '20'],
      ['New per day', '8'],
      ['Retention', '87%'],
      ['Budget unit', 'Reviews'],
      ['Learned when', 'By remembering'],
      ['Even load', 'Yes'],
    ])
    expect(createPresetText('Sanskrit', settings)).toBe(
      '---\ntype: preset\ntitle: Sanskrit\ngoal: minutes_a_day\nminutes_a_day: 20\nnew_a_day: 8\nretention: 0.87\n' +
        'counts: shows\nlearned: retention\neven_load: true\n---\n\n# Sanskrit\n',
    )
  })

  it('names the settings it knows when one is not a setting', () => {
    expect(() => parsePresetSettings([['Setting', 'Value'], ['Speed', '3']])).toThrow('"Speed" is not a setting of a preset: use Goal')
  })

  it('refuses a retention that is not a percentage, and a choice that is not offered', () => {
    expect(() => parsePresetSettings([['Setting', 'Value'], ['Retention', '0.87']])).toThrow('is not a percentage')
    expect(() => parsePresetSettings([['Setting', 'Value'], ['Budget unit', 'Words']])).toThrow(
      'the setting "Budget unit": "Words" is not one of Cards, Reviews',
    )
  })
})
