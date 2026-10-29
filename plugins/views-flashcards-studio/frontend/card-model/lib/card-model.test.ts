/** The effect of a card model gesture on the file, tested without a screen. */
import { describe, expect, it } from 'vitest'
import type { VaultCardModel } from '../../cards'
import {
  addTemplate,
  addField,
  createVaultTemplates,
  moveField,
  removeTemplate,
  removeField,
  renameTemplate,
  areCardModelsEqual,
  serializeCardModel,
  parseCardModel,
  createBufferCardModel,
  writeTemplate,
  type BufferCardModel,
} from './card-model'

/** Identities counted out, so a test names the template it means. */
const createIds = () => {
  let count = 0
  return () => `c${(count += 1)}`
}

const createTestCardModel = (overrides: Partial<VaultCardModel> = {}): VaultCardModel => ({
  path: 'Animal.md',
  title: 'Animal',
  fields: ['Height', 'Life span'],
  preamble: '',
  templates: [
    { name: 'Recognise', preamble: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
    { name: 'Name it', preamble: '', front: 'Which lives {{Life span}}?', back: '{{Height}}' },
  ],
  tail: '',
  problems: [],
  ...overrides,
})

const cardModel = (overrides: Partial<VaultCardModel> = {}): BufferCardModel =>
  createBufferCardModel(createTestCardModel(overrides), createIds())

describe('a card model as the window holds it', () => {
  it('gives every template an identity of its own', () => {
    expect(cardModel().templates.map((template) => template.id)).toStrictEqual(['c1', 'c2'])
  })

  it('is the same string read out and written back', () => {
    const target = cardModel()
    expect(parseCardModel(serializeCardModel(target))).toStrictEqual(target)
  })

  it('is a card model of no fields where nothing has been read', () => {
    expect(parseCardModel('')).toStrictEqual({ fields: [], preamble: '', templates: [], tail: '' })
  })

  it('passes the vault the templates without the identities it minted', () => {
    expect(createVaultTemplates(cardModel())[0]).toStrictEqual({
      name: 'Recognise',
      preamble: '',
      front: '{{Height}}',
      back: '**Height:** {{Height}}',
    })
  })
})

describe('a field of a card model', () => {
  it('is added at the end of the order', () => {
    expect(addField(cardModel(), 'Weight').fields).toStrictEqual(['Height', 'Life span', 'Weight'])
  })

  it('leaves the braces standing when the card model no longer names it', () => {
    const target = removeField(cardModel(), 'Height')
    expect(target.fields).toStrictEqual(['Life span'])
    expect(target.templates[0]?.back).toBe('**Height:** {{Height}}')
  })

  it('writes into that template alone', () => {
    const target = writeTemplate(cardModel(), 'c1', 'back', 'changed')
    expect(target.templates[0]?.back).toBe('changed')
    expect(target.templates[1]?.back).toBe('{{Height}}')
  })

  it('lands before the field it was let go on', () => {
    expect(
      moveField(cardModel({ fields: ['Name', 'Height', 'Life span'] }), 'Life span', 'Height')
        .fields,
    ).toStrictEqual(['Name', 'Life span', 'Height'])
  })

  it('leaves the first field first, wherever the move came from', () => {
    const target = cardModel({ fields: ['Name', 'Height', 'Life span'] })
    expect(moveField(target, 'Height', 'Name').fields).toStrictEqual(target.fields)
    expect(moveField(target, 'Name', null).fields).toStrictEqual(target.fields)
  })
})

describe('a template of a card model', () => {
  it('is added at the end, with both its halves empty', () => {
    const target = addTemplate(cardModel(), 'Spell it', () => 'c9')
    expect(target.templates[2]).toStrictEqual({
      id: 'c9',
      name: 'Spell it',
      preamble: '',
      front: '',
      back: '',
    })
  })

  it('takes the name it was given', () => {
    expect(renameTemplate(cardModel(), 'c1', 'Spot it').templates[0]?.name).toBe('Spot it')
  })

  it('renames that template alone', () => {
    expect(renameTemplate(cardModel(), 'c1', 'Spot it').templates[1]?.name).toBe('Name it')
  })

  it('leaves every template as it was when it names none', () => {
    expect(renameTemplate(cardModel(), 'nobody', 'Spot it')).toStrictEqual(cardModel())
    expect(writeTemplate(cardModel(), 'nobody', 'front', 'x')).toStrictEqual(cardModel())
  })

  it('goes, and the rest stay in the order they were in', () => {
    expect(
      removeTemplate(cardModel(), 'c1').templates.map((template) => template.name),
    ).toStrictEqual(['Name it'])
  })

  it('takes what was written into one half, and the other remains', () => {
    const target = writeTemplate(cardModel(), 'c1', 'front', '{{Height}}')
    expect(target.templates[0]?.front).toBe('{{Height}}')
    expect(target.templates[0]?.back).toBe('**Height:** {{Height}}')
  })
})

describe('whether two evaluations of a file read the same', () => {
  it('is so for one file read twice, whatever identities each reading minted', () => {
    expect(areCardModelsEqual(cardModel(), cardModel())).toBe(true)
  })

  it('is not so for a field or a template written elsewhere', () => {
    expect(areCardModelsEqual(cardModel(), cardModel({ fields: ['Height'] }))).toBe(false)
    expect(
      areCardModelsEqual(
        cardModel(),
        cardModel({ templates: createTestCardModel().templates.slice(0, 1) }),
      ),
    ).toBe(false)
  })

  it('is not so for a preamble written elsewhere, however the tail reads', () => {
    expect(areCardModelsEqual(cardModel(), cardModel({ preamble: 'Intro' }))).toBe(false)
  })

  it('is not so for a tail written elsewhere, however the preamble reads', () => {
    expect(areCardModelsEqual(cardModel(), cardModel({ tail: 'Outro' }))).toBe(false)
  })
})
