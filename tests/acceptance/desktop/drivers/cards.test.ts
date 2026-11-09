import { describe, expect, it } from 'vitest'
import {
  assertCardFileIsFree,
  assertCardFileKind,
  countCards,
  createCardId,
  createCardModelText,
  createCardText,
  createDeckText,
  createPresetText,
  createTemplatesText,
  getCardFile,
  parseCardModels,
  parseCards,
  parsePresets,
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

describe('parseCardModels', () => {
  const header = ['card model', 'fields', 'template', 'front', 'back']

  it('reads the fields of the first row and a template in each row', () => {
    expect(
      parseCardModels([
        header,
        ['Animal', 'Name, Height', 'Recognise', '{{Name}}', '{{Height}}'],
        ['Animal', '', 'Name it', '{{Name}}?', '{{Height}}'],
        ['Plant', 'Name', '', '', ''],
      ]),
    ).toEqual([
      {
        title: 'Animal',
        fields: ['Name', 'Height'],
        templates: [
          { name: 'Recognise', front: '{{Name}}', back: '{{Height}}' },
          { name: 'Name it', front: '{{Name}}?', back: '{{Height}}' },
        ],
      },
      { title: 'Plant', fields: ['Name'], templates: [] },
    ])
  })

  it.each([
    [[['Model', 'fields', 'template', 'front', 'back'], ['A', 'x', '', '', '']], 'the card models is a table with the header "card model", "fields"'],
    [[header], 'the card models has no row'],
    [[header, ['', 'x', '', '', '']], 'has no card model'],
    [[header, ['A', '', '', '', '']], 'the card model "A" has no fields: write them in its first row'],
    [[header, ['A', 'x,,y', '', '', '']], 'are names with commas between them'],
    [[header, ['A', 'x, x', '', '', '']], 'the fields x, x name one field twice'],
    [[header, ['A', 'x', 'T', 'f', 'b'], ['A', 'y', 'U', 'f', 'b']], 'has its fields in more than one row'],
    [[header, ['A', 'x', '', 'f', 'b']], 'a template of the card model "A" has no name'],
    [[header, ['A', 'x', 'T', 'f', 'b'], ['A', '', 'T', 'f', 'b']], 'the card model "A" has the template "T" twice'],
  ])('refuses %j', (rows, message) => {
    expect(() => parseCardModels(rows)).toThrow(message)
  })
})

describe('parseCards', () => {
  it('reads the field columns after the deck and the card model', () => {
    expect(
      parseCards([
        ['deck', 'card model', 'Name', 'Height'],
        ['zoo', 'Animal', 'Guanaco', ''],
        ['empty-deck', '', '', ''],
      ]),
    ).toEqual([
      { deck: 'zoo', model: 'Animal', values: [['Name', 'Guanaco'], ['Height', '']] },
      { deck: 'empty-deck', model: undefined, values: [['Name', ''], ['Height', '']] },
    ])
  })

  it.each([
    [[['card model', 'deck', 'Name'], ['Animal', 'zoo', 'x']], 'the cards are a table with the columns "deck" and "card model"'],
    [[['deck', 'card model', 'Name', 'Name'], ['zoo', 'Animal', 'x', 'y']], 'the cards has the column "Name" twice'],
    [[['deck', 'card model', '']], 'then one column for each field'],
    [[['deck', 'card model', 'Name']], 'the cards have no row'],
    [[['deck', 'card model', 'Name'], ['', 'Animal', 'x']], 'a card has no deck'],
    [[['deck', 'card model', 'Name'], ['zoo', '', 'x']], 'a card in the deck "zoo" has values and no card model'],
  ])('refuses %j', (rows, message) => {
    expect(() => parseCards(rows)).toThrow(message)
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

  it('reads the fields it wrote', () => {
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

  it('leaves out the fields without a value and refuses a card without any', () => {
    const card = createCardText('Animal', ['Name', 'Height'], [['Name', 'Llama'], ['Height', '']], id)
    expect(card).toBe(`\n## Llama ^${id}\n\n[[Animal]]\n\n### Name\n\nLlama\n`)
    expect(() => createCardText('Animal', ['Name'], [['Name', '']], id)).toThrow('a card of the card model "Animal" has no value')
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
    const [preset] = parsePresets([
      ['preset', 'Goal', 'Minutes per day', 'New per day', 'Retention', 'Budget unit', 'Learned when', 'Even load', 'Days apart'],
      ['Sanskrit', 'Minutes per day', '20', '8', '87%', 'Reviews', 'By remembering', 'Yes', ''],
    ])
    expect(preset.title).toBe('Sanskrit')
    expect(createPresetText(preset.title, preset.settings)).toBe(
      '---\ntype: preset\ntitle: Sanskrit\ngoal: minutes_a_day\nminutes_a_day: 20\nnew_a_day: 8\nretention: 0.87\n' +
        'counts: shows\nlearned: retention\neven_load: true\n---\n\n# Sanskrit\n',
    )
  })

  it('makes one preset for each row', () => {
    expect(parsePresets([['preset', 'New per day'], ['Quick', '10'], ['Slow', '']]).map((one) => one.settings)).toEqual([
      [['new_a_day', '10']],
      [],
    ])
  })

  it('names the settings it knows when a column is not a setting', () => {
    expect(() => parsePresets([['preset', 'Speed'], ['Quick', '3']])).toThrow('"Speed" is not a setting of a preset: use Goal')
  })

  it('refuses a table without the preset column, a column twice and a table without rows', () => {
    expect(() => parsePresets([['Goal'], ['Retention']])).toThrow('starts with the column "preset"')
    expect(() => parsePresets([['preset', 'Goal', 'Goal'], ['Quick', 'Retention', 'Retention']])).toThrow('has the column "Goal" twice')
    expect(() => parsePresets([['preset', 'Goal']])).toThrow('the presets have no row')
    expect(() => parsePresets([['preset', 'Goal'], ['', 'Retention']])).toThrow('a preset has no title')
  })

  it('refuses a retention that is not a percentage, and a choice that is not offered', () => {
    expect(() => parsePresets([['preset', 'Retention'], ['Quick', '0.87']])).toThrow('the setting "Retention": "0.87" is not a percentage')
    expect(() => parsePresets([['preset', 'Budget unit'], ['Quick', 'Words']])).toThrow(
      'the setting "Budget unit": "Words" is not one of Cards, Reviews',
    )
  })
})

describe('the rules of a card file', () => {
  const model = createCardModelText('Animal', ['Name'])
  const deck = createDeckText('Animal')

  it('refuses a second file with one title, a model and a deck alike', () => {
    expect(() => assertCardFileIsFree('Test', 'Animal', deck)).toThrow(
      'the vault Test already has cards/Animal.md: a card model, a deck and a preset need different titles',
    )
    expect(() => assertCardFileIsFree('Test', 'Animal', undefined)).not.toThrow()
  })

  it('gives the text of the file to add to', () => {
    expect(assertCardFileKind(model, 'Animal', 'card model')).toBe(model)
    expect(assertCardFileKind(deck, 'Animal', 'deck')).toBe(deck)
  })

  it('refuses a missing file', () => {
    expect(() => assertCardFileKind(undefined, 'Animal', 'card model')).toThrow(
      'the vault has no card model "Animal": make it with its fields first',
    )
    expect(() => assertCardFileKind(undefined, 'Zoo', 'deck')).toThrow('the vault has no deck "Zoo": make it first')
  })

  it('refuses a file of another type', () => {
    expect(() => assertCardFileKind(deck, 'Animal', 'card model')).toThrow('the vault has a deck "Animal", not a card model')
    expect(() => assertCardFileKind(model, 'Animal', 'deck')).toThrow('the vault has a card model "Animal", not a deck')
  })
})
