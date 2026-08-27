/**
 * What a gesture in a deck or a stencil makes of the file, asked without a
 * screen, and where each problem the vault reports is drawn.
 */
import { describe, expect, it } from 'vitest'
import type { Decked, Problem, Stencilled } from '../core'
import {
  added,
  bodyOf,
  cardsOf,
  carried,
  cutsOf,
  deckIn,
  deckOf,
  drawnOf,
  faceAdded,
  faceGone,
  faceNamed,
  faceWritten,
  facesOf,
  fieldAdded,
  fieldCarried,
  fieldGone,
  fieldNamed,
  filled,
  marksOf,
  named,
  pathOfCut,
  removed,
  sheetBodyOf,
  sheetIn,
  sheetOf,
  standingIn,
  type Deck,
  type Sheet,
} from './model'

/** Identities counted out, so a test names the card it means. */
const minting = () => {
  let at = 0
  return () => `c${(at += 1)}`
}

const read = (over: Partial<Decked> = {}): Decked => ({
  path: 'Animals.md',
  title: 'Animals',
  preamble: 'about the animals\n',
  cards: [
    {
      name: 'Llama',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: '',
      values: [
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    },
    {
      name: 'Alpaca',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: 'a note in the middle\n',
      values: [],
    },
  ],
  tail: '\n',
  problems: [],
  ...over,
})

const deck = (over: Partial<Decked> = {}): Deck => deckOf(read(over), minting())

const cut = (over: Partial<Stencilled> = {}): Stencilled => ({
  path: 'Animal.md',
  title: 'Animal',
  fields: ['Height', 'Life span'],
  faces: [
    { name: 'Recognise', front: '{{title}}', back: '**Height:** {{Height}}' },
    { name: 'Name it', front: 'Which lives {{Life span}}?', back: '{{title}}' },
  ],
  problems: [],
  ...over,
})

const sheet = (over: Partial<Stencilled> = {}): Sheet => sheetOf(cut(over), minting())

const problem = (over: Partial<Problem> = {}): Problem => ({
  fault: 'cardWithoutAName',
  card: null,
  face: null,
  field: '',
  text: 'a card with no name',
  ...over,
})

describe('a deck as the window holds it', () => {
  it('gives every card an identity of its own', () => {
    expect(deck().cards.map((card) => card.id)).toStrictEqual(['c1', 'c2'])
  })

  it('keeps the preamble, the tail and each card’s lead as the file had them', () => {
    const held = deck()
    expect(held.preamble).toBe('about the animals\n')
    expect(held.tail).toBe('\n')
    expect(held.cards[1]?.lead).toBe('a note in the middle\n')
  })

  it('is the same string read out and written back', () => {
    const held = deck()
    expect(deckIn(bodyOf(held))).toStrictEqual(held)
  })

  it('is a deck of no cards where nothing has been read', () => {
    expect(deckIn('')).toStrictEqual({ preamble: '', cards: [], tail: '' })
  })

  it('hands the vault the cards without the identities it minted', () => {
    expect(cardsOf(deck())[0]).toStrictEqual({
      name: 'Llama',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: '',
      values: [
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    })
  })
})

describe('the cards as the grid draws them', () => {
  const OFFERS = [
    { path: 'stencils/Animal.md', title: 'Animal', fields: ['Height', 'Life span'] },
  ]

  /** One card, under the wikilink it wrote and the stencil that link reached. */
  const cutBy = (stencil: string, stencilAt: string): Deck =>
    deck({ cards: [{ name: 'Llama', stencil, stencilAt, lead: '', values: [] }] })

  it('carries the name, what cuts it, and the values under it', () => {
    expect(drawnOf(deck(), OFFERS)[0]).toStrictEqual({
      id: 'c1',
      name: 'Llama',
      stencil: 'Animal',
      filled: [
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    })
  })

  it('says nothing of the lead, which nothing lays out', () => {
    expect(JSON.stringify(drawnOf(deck(), OFFERS))).not.toContain('a note in the middle')
  })

  it('draws a card the stencil its link reached, whatever stood in the brackets', () => {
    const written = ['cards/Animal', 'Animal|зверь', 'Animal#Recognise', 'stencils/Animal.md']
    for (const one of written) {
      expect(drawnOf(cutBy(one, 'stencils/Animal.md'), OFFERS)[0]?.stencil).toBe('Animal')
    }
  })

  it('draws a card whose link reached nothing under what the file wrote', () => {
    expect(drawnOf(cutBy('Gone', ''), OFFERS)[0]?.stencil).toBe('Gone')
  })
})

describe('the stencils a card may be cut by', () => {
  it('are named by what each stencil is called', () => {
    expect(
      cutsOf([
        { path: 'stencils/Animal.md', title: 'Animal', fields: ['Height'] },
        { path: 'Term.md', title: 'Term', fields: ['Meaning'] },
      ]),
    ).toStrictEqual([
      { name: 'Animal', fields: ['Height'] },
      { name: 'Term', fields: ['Meaning'] },
    ])
  })

  it('name one stencil where two are called one name, and the first stands', () => {
    expect(
      cutsOf([
        { path: 'a/Animal.md', title: 'Animal', fields: ['Height'] },
        { path: 'b/Animal.md', title: 'Animal', fields: ['Weight'] },
      ]),
    ).toStrictEqual([{ name: 'Animal', fields: ['Height'] }])
  })

  it('leave out a stencil called nothing', () => {
    expect(cutsOf([{ path: 'Animal.md', title: '', fields: [] }])).toStrictEqual([])
  })

  it('are filed where the list said, and nowhere for a name no stencil carries', () => {
    const offers = [{ path: 'stencils/Animal.md', title: 'Animal', fields: [] }]
    expect(pathOfCut(offers, 'Animal')).toBe('stencils/Animal.md')
    expect(pathOfCut(offers, 'Term')).toBe('')
  })
})

describe('a card added', () => {
  it('stands last, cut by the stencil it was asked for', () => {
    const held = added(
      deck(),
      'Vicuña',
      'Animal',
      'stencils/Animal.md',
      [{ field: 'Height', text: '' }],
      () => 'c9',
    )
    expect(held.cards.map((card) => card.name)).toStrictEqual(['Llama', 'Alpaca', 'Vicuña'])
    // What the card is named by stands in the heading, and in no value.
    expect(held.cards[2]).toStrictEqual({
      id: 'c9',
      name: 'Vicuña',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: '',
      values: [{ field: 'Height', text: '' }],
    })
  })

  it('leaves the preamble and the tail where they were', () => {
    const held = added(deck(), 'Vicuña', 'Animal', 'stencils/Animal.md', [], () => 'c9')
    expect(held.preamble).toBe('about the animals\n')
    expect(held.tail).toBe('\n')
  })
})

describe('a card taken out, renamed and carried', () => {
  it('goes, and the rest stay in the order they were in', () => {
    expect(removed(deck(), 'c1').cards.map((card) => card.name)).toStrictEqual(['Alpaca'])
  })

  it('is nothing for an identity the deck does not hold', () => {
    expect(removed(deck(), 'c9').cards).toHaveLength(2)
  })

  it('takes the name it was given, and no other card takes it', () => {
    const held = named(deck(), 'c2', 'Vicuña')
    expect(held.cards.map((card) => card.name)).toStrictEqual(['Llama', 'Vicuña'])
  })

  it('lands before the card it was let go on', () => {
    expect(carried(deck(), 'c2', 'c1').cards.map((card) => card.name)).toStrictEqual([
      'Alpaca',
      'Llama',
    ])
  })

  it('lands last where it was let go on nothing', () => {
    expect(carried(deck(), 'c1', null).cards.map((card) => card.name)).toStrictEqual([
      'Alpaca',
      'Llama',
    ])
  })
})

describe('a value written into a card', () => {
  it('stands where the field already was', () => {
    const held = filled(deck(), 'c1', 'Height', 'about 46"')
    expect(held.cards[0]?.values).toStrictEqual([
      { field: 'Height', text: 'about 46"' },
      { field: 'Life span', text: 'about 20 years' },
    ])
  })

  it('is written after the rest where the card had no such field', () => {
    const held = filled(deck(), 'c1', 'Weight', '130 kg')
    expect(held.cards[0]?.values.map((value) => value.field)).toStrictEqual([
      'Height',
      'Life span',
      'Weight',
    ])
  })

  it('keeps the heading of a field the card has, emptied', () => {
    const held = filled(deck(), 'c1', 'Height', '')
    expect(held.cards[0]?.values[0]).toStrictEqual({ field: 'Height', text: '' })
  })

  it('writes no heading for a field the card does not have and nothing was typed into', () => {
    expect(filled(deck(), 'c2', 'Height', '').cards[1]?.values).toStrictEqual([])
  })

  it('leaves every other card as it was', () => {
    expect(filled(deck(), 'c1', 'Height', 'taller').cards[1]).toStrictEqual(
      deck().cards[1],
    )
  })
})

describe('a stencil as the window holds it', () => {
  it('gives every face an identity of its own', () => {
    expect(sheet().faces.map((face) => face.id)).toStrictEqual(['c1', 'c2'])
  })

  it('is the same string read out and written back', () => {
    const held = sheet()
    expect(sheetIn(sheetBodyOf(held))).toStrictEqual(held)
  })

  it('is a stencil of no fields where nothing has been read', () => {
    expect(sheetIn('')).toStrictEqual({ fields: [], faces: [] })
  })

  it('hands the vault the faces without the identities it minted', () => {
    expect(facesOf(sheet())[0]).toStrictEqual({
      name: 'Recognise',
      front: '{{title}}',
      back: '**Height:** {{Height}}',
    })
  })
})

describe('a field of a stencil', () => {
  it('is added at the end of the order', () => {
    expect(fieldAdded(sheet(), 'Weight').fields).toStrictEqual([
      'Height',
      'Life span',
      'Weight',
    ])
  })

  it('is renamed in the braces of every face that stands it', () => {
    const held = fieldNamed(sheet(), 'Height', 'Shoulder')
    expect(held.fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(held.faces[0]?.back).toBe('**Height:** {{Shoulder}}')
  })

  it('leaves the braces of every other field where they are', () => {
    expect(fieldNamed(sheet(), 'Height', 'Shoulder').faces[1]?.front).toBe(
      'Which lives {{Life span}}?',
    )
  })

  it('leaves the braces standing when the stencil no longer names it', () => {
    const held = fieldGone(sheet(), 'Height')
    expect(held.fields).toStrictEqual(['Life span'])
    expect(held.faces[0]?.back).toBe('**Height:** {{Height}}')
  })

  it('lands before the field it was let go on', () => {
    expect(fieldCarried(sheet(), 'Life span', 'Height').fields).toStrictEqual([
      'Life span',
      'Height',
    ])
  })
})

describe('a face of a stencil', () => {
  it('is added at the end, with both its halves empty', () => {
    const held = faceAdded(sheet(), 'Spell it', () => 'c9')
    expect(held.faces[2]).toStrictEqual({ id: 'c9', name: 'Spell it', front: '', back: '' })
  })

  it('takes the name it was given', () => {
    expect(faceNamed(sheet(), 'c1', 'Spot it').faces[0]?.name).toBe('Spot it')
  })

  it('goes, and the rest stay in the order they were in', () => {
    expect(faceGone(sheet(), 'c1').faces.map((face) => face.name)).toStrictEqual(['Name it'])
  })

  it('takes what was written into one half, and the other stands', () => {
    const held = faceWritten(sheet(), 'c1', 'front', '{{Height}}')
    expect(held.faces[0]?.front).toBe('{{Height}}')
    expect(held.faces[0]?.back).toBe('**Height:** {{Height}}')
  })
})

describe('where a problem is drawn', () => {
  it('is the card it was read against, counted from the first', () => {
    const marks = marksOf([problem({ card: 1 })], ['c1', 'c2'], [])
    expect(marks.at.get('c2')).toStrictEqual(['a card with no name'])
    expect(marks.at.has('c1')).toBe(false)
  })

  it('is the card and not the name, so two cards of one name are told apart', () => {
    const marks = marksOf(
      [
        problem({ fault: 'cardNamedTwice', card: 1, text: 'named twice' }),
        problem({ fault: 'cardNamedTwice', card: 2, text: 'named twice' }),
      ],
      ['c1', 'c2', 'c3'],
      [],
    )
    expect([...marks.at.keys()]).toStrictEqual(['c2', 'c3'])
  })

  it('stands every problem of one card together, in the order they were read', () => {
    const marks = marksOf(
      [problem({ card: 0, text: 'first' }), problem({ card: 0, text: 'second' })],
      ['c1'],
      [],
    )
    expect(marks.at.get('c1')).toStrictEqual(['first', 'second'])
  })

  it('is the face it was read against', () => {
    const marks = marksOf(
      [problem({ fault: 'faceMissingASide', face: 0, text: 'no back' })],
      [],
      ['f1', 'f2'],
    )
    expect(marks.at.get('f1')).toStrictEqual(['no back'])
  })

  it('is the field where it stands against no card and no face', () => {
    const marks = marksOf(
      [problem({ fault: 'fieldDeclaredTwice', field: 'Height', text: 'declared twice' })],
      [],
      [],
    )
    expect(marks.fields.get('Height')).toStrictEqual(['declared twice'])
    expect(marks.whole).toStrictEqual([])
  })

  it('is the file where it stands against none of the three', () => {
    const marks = marksOf([problem({ text: 'something else' })], ['c1'], [])
    expect(marks.whole).toStrictEqual(['something else'])
  })

  it('is the file where the card it names was not read', () => {
    expect(marksOf([problem({ card: 5 })], ['c1'], []).whole).toStrictEqual([
      'a card with no name',
    ])
  })
})

describe('where a mark is drawn', () => {
  it('is the one element carrying that value', () => {
    expect(standingIn('data-card', 'c1')).toBe('[data-card="c1"]')
  })

  it('writes out a quote and a backslash, which a name may carry', () => {
    expect(standingIn('data-field', 'a "wide" one')).toBe('[data-field="a \\"wide\\" one"]')
    expect(standingIn('data-field', 'a\\b')).toBe('[data-field="a\\\\b"]')
  })
})
