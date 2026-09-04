/**
 * What a gesture in a deck or a stencil makes of the file, asked without a
 * screen, and where each problem the vault reports is drawn.
 */
import { describe, expect, it } from 'vitest'
import { CARD_HEAD, cardEndOf as endOfCards } from '@numen/ui'
import type { Decked, Problem, Stencilled } from '../core'
import {
  added,
  bodyOf,
  cardsOf,
  carried,
  stencilsOf,
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
  filled,
  linkTo,
  marksOf,
  pathOfCut,
  removed,
  sameDeck,
  sameSheet,
  sectionAdded,
  sectionGone,
  sectionNamed,
  sectionsOf,
  sheetBodyOf,
  sheetIn,
  sheetOf,
  standingIn,
  type Deck,
  type Sheet,
} from './body'

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
      mark: 'k7m2xq9fzp',
      section: null,
      heading: 'Llama',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: '',
      values: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    },
    {
      mark: '3n8vr4tqch',
      section: null,
      heading: 'Alpaca',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: 'a note in the middle\n',
      values: [],
    },
  ],
  sections: [],
  tail: '\n',
  problems: [],
  ...over,
})

/** The two cards of the fixture, by the mark each carries. */
const LLAMA = 'k7m2xq9fzp'
const ALPACA = '3n8vr4tqch'

const deck = (over: Partial<Decked> = {}): Deck => deckOf(read(over), minting())

const cut = (over: Partial<Stencilled> = {}): Stencilled => ({
  path: 'Animal.md',
  title: 'Animal',
  fields: ['Height', 'Life span'],
  preamble: '',
  faces: [
    { name: 'Recognise', lead: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
    { name: 'Name it', lead: '', front: 'Which lives {{Life span}}?', back: '{{Height}}' },
  ],
  tail: '',
  problems: [],
  ...over,
})

const sheet = (over: Partial<Stencilled> = {}): Sheet => sheetOf(cut(over), minting())

const problem = (over: Partial<Problem> = {}): Problem => ({
  fault: 'cardWithoutAStencil',
  card: null,
  face: null,
  field: '',
  text: 'a card under no stencil',
  ...over,
})

describe('a deck as the window holds it', () => {
  it('knows every card by the mark the file carries for it', () => {
    expect(deck().cards.map((card) => card.id)).toStrictEqual([LLAMA, ALPACA])
  })

  it('mints an identity for a card carrying no mark, which the file cannot name', () => {
    const bare = read()
    const held = deck({
      cards: [{ ...bare.cards[0]!, mark: '' }, { ...bare.cards[1]!, mark: '' }],
    })
    expect(held.cards.map((card) => card.id)).toStrictEqual(['c1', 'c2'])
  })

  it('mints one for each of two cards carrying one mark, so both are drawn apart', () => {
    const bare = read()
    const held = deck({ cards: [bare.cards[0]!, { ...bare.cards[1]!, mark: LLAMA }] })
    expect(held.cards.map((card) => card.id)).toStrictEqual(['c1', 'c2'])
  })

  it('gives every section an identity of its own, no file naming one', () => {
    const held = deck({ sections: [{ name: 'Roots', lead: '' }] })
    expect(held.sections.map((section) => section.id)).toStrictEqual(['c1'])
  })

  it('stands each card under the section the file put it under', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', lead: '' }],
      cards: [{ ...bare.cards[0]!, section: 0 }, bare.cards[1]!],
    })
    expect(held.cards.map((card) => card.section)).toStrictEqual([held.sections[0]?.id, null])
  })

  /* A section the reading does not hold is no section: the card stands before
     the first, drawn and counted, and is written back standing there. */
  it('stands a card before the first section where the reading holds no section it names', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', lead: '' }],
      cards: [{ ...bare.cards[0]!, section: 7 }, bare.cards[1]!],
    })
    expect(held.cards.map((card) => card.section)).toStrictEqual([null, null])
    expect(cardsOf(held).map((card) => card.section)).toStrictEqual([null, null])
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
    expect(deckIn('')).toStrictEqual({ preamble: '', cards: [], sections: [], tail: '' })
  })

  it('hands the vault the cards without the identities it minted', () => {
    expect(cardsOf(deck())[0]).toStrictEqual({
      mark: LLAMA,
      section: null,
      heading: 'Llama',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: '',
      values: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    })
  })

  it('hands the vault each card under where its section stands among them', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', lead: '' }, { name: 'Leaves', lead: '' }],
      cards: [{ ...bare.cards[0]!, section: 1 }, bare.cards[1]!],
    })
    expect(cardsOf(held).map((card) => card.section)).toStrictEqual([1, null])
  })

  it('hands the vault the sections without the identities it minted', () => {
    const held = deck({ sections: [{ name: 'Roots', lead: 'about the roots\n' }] })
    expect(sectionsOf(held)).toStrictEqual([{ name: 'Roots', lead: 'about the roots\n' }])
  })
})

describe('the cards as the grid draws them', () => {
  const OFFERS = [
    { path: 'stencils/Animal.md', title: 'Animal', fields: ['Height', 'Life span'] },
  ]

  /** One card, under the wikilink it wrote and the stencil that link reached. */
  const cutBy = (stencil: string, stencilAt: string): Deck =>
    deck({
      cards: [
        {
          mark: LLAMA,
          section: null,
          heading: 'Llama',
          stencil,
          stencilAt,
          lead: '',
          values: [],
        },
      ],
    })

  it('carries the identity, the section, what cuts it, and the values under it', () => {
    expect(drawnOf(deck(), OFFERS)[0]).toStrictEqual({
      id: LLAMA,
      section: null,
      stencil: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    })
  })

  it('says nothing of the lead, which nothing lays out', () => {
    expect(JSON.stringify(drawnOf(deck(), OFFERS))).not.toContain('a note in the middle')
  })

  it('draws a card the stencil its link reached, whatever stood in the brackets', () => {
    const written = ['cards/Animal', 'Animal|животное', 'Animal#Recognise', 'stencils/Animal.md']
    for (const one of written) {
      expect(drawnOf(cutBy(one, 'stencils/Animal.md'), OFFERS)[0]?.stencil).toBe('Animal')
    }
  })

  it('draws a card whose link reached nothing under what the file wrote', () => {
    expect(drawnOf(cutBy('Gone', ''), OFFERS)[0]?.stencil).toBe('Gone')
  })

  it('draws a card that wrote no brackets at all as cut by nothing', () => {
    expect(drawnOf(cutBy('', ''), OFFERS)[0]?.stencil).toBeNull()
  })
})

describe('the stencils a card may be cut by', () => {
  it('are named by what each stencil is called', () => {
    expect(
      stencilsOf([
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
      stencilsOf([
        { path: 'a/Animal.md', title: 'Animal', fields: ['Height'] },
        { path: 'b/Animal.md', title: 'Animal', fields: ['Weight'] },
      ]),
    ).toStrictEqual([{ name: 'Animal', fields: ['Height'] }])
  })

  it('leave out a stencil called nothing', () => {
    expect(stencilsOf([{ path: 'Animal.md', title: '', fields: [] }])).toStrictEqual([])
  })

  it('are filed where the list said, and nowhere for a name no stencil carries', () => {
    const offers = [{ path: 'stencils/Animal.md', title: 'Animal', fields: [] }]
    expect(pathOfCut(offers, 'Animal')).toBe('stencils/Animal.md')
    expect(pathOfCut(offers, 'Term')).toBe('')
  })
})

describe('a card added', () => {
  it('stands last, cut by the stencil it was asked for and carrying no mark', () => {
    const held = added(
      deck(),
      'Animal',
      'stencils/Animal.md',
      [{ field: 'Name', text: '' }],
      null,
      () => 'c9',
    )
    expect(held.cards.map((card) => card.id)).toStrictEqual([LLAMA, ALPACA, 'c9'])
    // A mark is written where the deck is made whole, on its way to the vault.
    expect(held.cards[2]).toStrictEqual({
      id: 'c9',
      mark: '',
      section: null,
      heading: '',
      stencil: 'Animal',
      stencilAt: 'stencils/Animal.md',
      lead: '',
      values: [{ field: 'Name', text: '' }],
    })
  })

  it('stands under the section it was asked for, at the end of it', () => {
    const start = deck({ sections: [{ name: 'Roots', lead: '' }, { name: 'Leaves', lead: '' }] })
    const held = added(start, 'Animal', 'stencils/Animal.md', [], start.sections[1]!.id, () => 'c9')
    expect(held.cards[2]?.section).toBe(start.sections[1]?.id)
  })

  it('stands before the first section where it was asked for none', () => {
    const start = deck({ sections: [{ name: 'Roots', lead: '' }, { name: 'Leaves', lead: '' }] })
    const held = added(start, 'Animal', 'stencils/Animal.md', [], null, () => 'c9')
    expect(held.cards.find((card) => card.id === 'c9')?.section).toBe(null)
  })

  it('leaves the preamble and the tail where they were', () => {
    const held = added(deck(), 'Animal', 'stencils/Animal.md', [], null, () => 'c9')
    expect(held.preamble).toBe('about the animals\n')
    expect(held.tail).toBe('\n')
  })

  it('names its stencil by the file, where the stencil is titled another way', () => {
    const held = added(deck(), 'Animal', 'stencils/creature sheet.md', [], null, () => 'c9')
    expect(held.cards[2]?.stencil).toBe('creature sheet')
  })

  it('names its stencil by no title, which a link resolves by nowhere', () => {
    const held = added(deck(), 'Animal', 'stencils/creature sheet.md', [], null, () => 'c9')
    expect(held.cards[2]?.stencil).not.toBe('Animal')
  })

  it('names it by the title where the vault filed the stencil nowhere', () => {
    const held = added(deck(), 'Animal', '', [], null, () => 'c9')
    expect(held.cards[2]?.stencil).toBe('Animal')
  })
})

describe('how a card names the stencil it is cut by', () => {
  it('is the file, without the folders above it and without the extension', () => {
    expect(linkTo('stencils/cards/Animal.md')).toBe('Animal')
  })

  it('keeps a name a dot stands inside, and one carrying no extension at all', () => {
    expect(linkTo('stencils/Animal v2.md')).toBe('Animal v2')
    expect(linkTo('stencils/Animal')).toBe('Animal')
  })

  it('is nothing for a stencil filed nowhere', () => {
    expect(linkTo('')).toBe('')
  })
})

describe('a card taken out and carried', () => {
  it('goes, and the rest stay in the order they were in', () => {
    expect(removed(deck(), LLAMA).cards.map((card) => card.id)).toStrictEqual([ALPACA])
  })

  it('is nothing for an identity the deck does not hold', () => {
    expect(removed(deck(), 'c9').cards).toHaveLength(2)
  })

  it('lands before the card it was let go on', () => {
    expect(carried(deck(), ALPACA, LLAMA).cards.map((card) => card.id)).toStrictEqual([
      ALPACA,
      LLAMA,
    ])
  })

  it('lands last where it was let go on nothing', () => {
    expect(carried(deck(), LLAMA, null).cards.map((card) => card.id)).toStrictEqual([
      ALPACA,
      LLAMA,
    ])
  })
})

describe('a card carried among the sections', () => {
  /** Two sections, the first card in the first of them and the second in neither. */
  const sectioned = (): Deck => {
    const bare = read()
    return deck({
      sections: [{ name: 'Roots', lead: '' }, { name: 'Leaves', lead: '' }],
      cards: [bare.cards[0]!, { ...bare.cards[1]!, section: 0 }],
    })
  }

  it('takes the section of the card it was let go before', () => {
    const held = sectioned()
    const moved = carried(held, LLAMA, ALPACA)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, held.sections[0]?.id],
      [ALPACA, held.sections[0]?.id],
    ])
  })

  it('stands under the last section where it was let go on nothing', () => {
    const held = sectioned()
    expect(carried(held, LLAMA, null).cards.map((card) => card.section)).toStrictEqual([
      held.sections[0]?.id,
      held.sections[1]?.id,
    ])
  })

  it('lands at the head of a section it was let go on', () => {
    const held = sectioned()
    const roots = held.sections[0]?.id ?? ''
    const moved = carried(held, LLAMA, roots)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, roots],
      [ALPACA, roots],
    ])
  })

  it('lands at the head of a section holding no card', () => {
    const held = sectioned()
    const leaves = held.sections[1]?.id ?? ''
    const moved = carried(held, LLAMA, leaves)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, held.sections[0]?.id],
      [LLAMA, leaves],
    ])
  })

  it('moves nothing where it was let go on nothing the deck holds', () => {
    const held = sectioned()
    expect(carried(held, LLAMA, 'nowhere')).toStrictEqual(held)
  })

  it('stands last under the section it was let go past the end of', () => {
    const held = sectioned()
    const roots = held.sections[0]?.id ?? ''
    const moved = carried(held, LLAMA, endOfCards(roots))
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, roots],
      [LLAMA, roots],
    ])
  })

  it('stands last under no section where it was let go past those before the first', () => {
    const held = sectioned()
    const moved = carried(held, ALPACA, endOfCards(CARD_HEAD))
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, null],
      [ALPACA, null],
    ])
  })

  // A card naming a section the deck does not hold is drawn before the first
  // heading, so that is where it counts from when another lands past it.
  it('stands past a card whose section the deck has lost, which stands under none', () => {
    const held = sectioned()
    const lost = {
      ...held.cards[0]!,
      section: 'gone',
    }
    const deckLost = { ...held, cards: [lost, held.cards[1]!] }

    const moved = carried(deckLost, ALPACA, endOfCards(CARD_HEAD))

    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, 'gone'],
      [ALPACA, null],
    ])
  })

  it('leaves the deck as it was where the card already stands last under that heading', () => {
    const held = sectioned()
    expect(carried(held, ALPACA, endOfCards(held.sections[0]?.id ?? ''))).toStrictEqual(held)
  })

  it('stands first and under no section where it was let go at the head of the deck', () => {
    const held = sectioned()
    const moved = carried(held, ALPACA, CARD_HEAD)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, null],
      [LLAMA, null],
    ])
  })
})

describe('a section of a deck', () => {
  const sectioned = (): Deck =>
    deck({ sections: [{ name: 'Roots', lead: '' }, { name: 'Leaves', lead: '' }] })

  it('is made at the end of the deck, holding no card', () => {
    const held = sectionAdded(sectioned(), 'Shoots', () => 'c9')
    expect(held.sections[2]).toStrictEqual({ id: 'c9', name: 'Shoots', lead: '' })
    expect(held.cards.map((card) => card.section)).toStrictEqual([null, null])
  })

  it('takes the name it was given, and no other section takes it', () => {
    const held = sectioned()
    const named = sectionNamed(held, held.sections[0]?.id ?? '', 'Roots and shoots')
    expect(named.sections.map((section) => section.name)).toStrictEqual([
      'Roots and shoots',
      'Leaves',
    ])
  })

  it('takes a name the section beside it carries, two being free to share one', () => {
    const held = sectioned()
    const named = sectionNamed(held, held.sections[0]?.id ?? '', 'Leaves')
    expect(named.sections.map((section) => section.name)).toStrictEqual(['Leaves', 'Leaves'])
  })

  /* Taking a section away takes away its heading and nothing else: its cards
     stay where they stand, under whatever heading is above them now. */
  it('leaves its cards under the section above it when it goes', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', lead: '' }, { name: 'Leaves', lead: '' }],
      cards: [{ ...bare.cards[0]!, section: 0 }, { ...bare.cards[1]!, section: 1 }],
    })
    const gone = sectionGone(held, held.sections[1]?.id ?? '')
    expect(gone.sections.map((section) => section.name)).toStrictEqual(['Roots'])
    expect(gone.cards.map((card) => card.section)).toStrictEqual([
      held.sections[0]?.id,
      held.sections[0]?.id,
    ])
  })

  /* A section is a name, so taking it away takes away a name: the text it stood
     on stays in the deck, standing after the text above it. */
  it('leaves the text it stood on with the section above it when it goes', () => {
    const held = deck({
      sections: [
        { name: 'Roots', lead: 'about the roots\n' },
        { name: 'Leaves', lead: 'about the leaves\n' },
      ],
    })
    const gone = sectionGone(held, held.sections[1]?.id ?? '')
    expect(gone.sections.map((section) => section.lead)).toStrictEqual([
      'about the roots\n\nabout the leaves\n',
    ])
  })

  it('leaves it with the deck’s own text where no section stands above it', () => {
    const held = deck({ sections: [{ name: 'Roots', lead: 'about the roots\n' }] })
    const gone = sectionGone(held, held.sections[0]?.id ?? '')
    expect(gone.preamble).toBe('about the animals\n\nabout the roots\n')
  })

  it('leaves the text above it alone where it stood on none of its own', () => {
    const held = deck({
      sections: [{ name: 'Roots', lead: 'about the roots\n' }, { name: 'Leaves', lead: '' }],
    })
    const gone = sectionGone(held, held.sections[1]?.id ?? '')
    expect(gone.sections.map((section) => section.lead)).toStrictEqual(['about the roots\n'])
  })

  it('leaves its cards under no section when the first of them goes', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', lead: '' }],
      cards: [{ ...bare.cards[0]!, section: 0 }, bare.cards[1]!],
    })
    const gone = sectionGone(held, held.sections[0]?.id ?? '')
    expect(gone.sections).toStrictEqual([])
    expect(gone.cards.map((card) => card.section)).toStrictEqual([null, null])
  })
})

describe('a value written into a card', () => {
  it('stands where the field already was', () => {
    const held = filled(deck(), LLAMA, 'Height', 1, 'about 46"')
    expect(held.cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 46"' },
      { field: 'Life span', text: 'about 20 years' },
    ])
  })

  it('is the one of two under a field that was typed in, and the other stands', () => {
    const twice = deck()
    const card = twice.cards[0]
    if (!card) throw new Error('the fixture holds no card')
    const held = filled(
      {
        ...twice,
        cards: [{ ...card, values: [...card.values, { field: 'Height', text: 'about 46"' }] }],
      },
      LLAMA,
      'Height',
      2,
      'about 47"',
    )
    expect(held.cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
      { field: 'Life span', text: 'about 20 years' },
      { field: 'Height', text: 'about 47"' },
    ])
  })

  it('is written after the rest where the card had no such field', () => {
    const held = filled(deck(), LLAMA, 'Weight', 1, '130 kg')
    expect(held.cards[0]?.values.map((value) => value.field)).toStrictEqual([
      'Name',
      'Height',
      'Life span',
      'Weight',
    ])
  })

  it('keeps the heading of a field the card has, emptied', () => {
    const held = filled(deck(), LLAMA, 'Height', 1, '')
    expect(held.cards[0]?.values[1]).toStrictEqual({ field: 'Height', text: '' })
  })

  it('writes no heading for a field the card does not have and nothing was typed into', () => {
    expect(filled(deck(), ALPACA, 'Height', 1, '').cards[1]?.values).toStrictEqual([])
  })

  it('leaves every other card as it was', () => {
    expect(filled(deck(), LLAMA, 'Height', 1, 'taller').cards[1]).toStrictEqual(
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
    expect(sheetIn('')).toStrictEqual({ fields: [], preamble: '', faces: [], tail: '' })
  })

  it('hands the vault the faces without the identities it minted', () => {
    expect(facesOf(sheet())[0]).toStrictEqual({
      name: 'Recognise',
      lead: '',
      front: '{{Height}}',
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

  it('leaves the braces standing when the stencil no longer names it', () => {
    const held = fieldGone(sheet(), 'Height')
    expect(held.fields).toStrictEqual(['Life span'])
    expect(held.faces[0]?.back).toBe('**Height:** {{Height}}')
  })

  it('lands before the field it was let go on', () => {
    expect(fieldCarried(sheet({ fields: ['Name', 'Height', 'Life span'] }), 'Life span', 'Height')
      .fields).toStrictEqual(['Name', 'Life span', 'Height'])
  })

  it('leaves the first field first, wherever the move came from', () => {
    const held = sheet({ fields: ['Name', 'Height', 'Life span'] })
    expect(fieldCarried(held, 'Height', 'Name').fields).toStrictEqual(held.fields)
    expect(fieldCarried(held, 'Name', null).fields).toStrictEqual(held.fields)
  })
})

describe('a face of a stencil', () => {
  it('is added at the end, with both its halves empty', () => {
    const held = faceAdded(sheet(), 'Spell it', () => 'c9')
    expect(held.faces[2]).toStrictEqual({
      id: 'c9',
      name: 'Spell it',
      lead: '',
      front: '',
      back: '',
    })
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
    expect(marks.at.get('c2')).toStrictEqual(['a card under no stencil'])
    expect(marks.at.has('c1')).toBe(false)
  })

  it('is the identity the card was drawn under, so two of one mark are told apart', () => {
    const marks = marksOf(
      [
        problem({ fault: 'markCarriedTwice', card: 1, text: 'a mark carried twice' }),
        problem({ fault: 'markCarriedTwice', card: 2, text: 'a mark carried twice' }),
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

  it('is the field of the card where it names both, so it is said under that value', () => {
    const marks = marksOf(
      [problem({ fault: 'fieldWrittenTwice', card: 1, field: 'Name', text: 'twice' })],
      ['c1', 'c2'],
      [],
    )
    expect(marks.under.get('c2')?.get('Name')).toStrictEqual(['twice'])
    expect(marks.at.has('c2')).toBe(false)
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
      'a card under no stencil',
    ])
  })
})

describe('whether two readings of a file read the same', () => {
  it('is so for one file read twice, whatever identities each reading minted', () => {
    expect(sameDeck(deck(), deck())).toBe(true)
    expect(sameSheet(sheet(), sheet())).toBe(true)
  })

  it('is not so for a card whose text was written elsewhere', () => {
    const other = read()
    const wrote = {
      ...other,
      cards: [{ ...other.cards[0]!, values: [{ field: 'Height', text: 'about 6ft' }] }],
    }

    expect(sameDeck(deck(), deckOf(wrote, minting()))).toBe(false)
  })

  // The heading is read back from the first field wherever the deck is written,
  // so a save carrying a value the window typed comes back under a heading the
  // window never held. That is the vault keeping its own word, not somebody
  // writing this deck elsewhere, and what a person is typing into stands.
  it('is so where only the heading a write read back is different', () => {
    const other = read()
    const wrote = {
      ...other,
      cards: [{ ...other.cards[0]!, heading: 'about 45"' }, ...other.cards.slice(1)],
    }

    expect(sameDeck(deck(), deckOf(wrote, minting()))).toBe(true)
  })

  it('is not so for prose around the cards written elsewhere', () => {
    expect(sameDeck(deck(), deck({ preamble: 'about them\n' }))).toBe(false)
    expect(sameDeck(deck(), deck({ tail: '\n\n' }))).toBe(false)
  })

  it('is not so for a card taken out, the ones left over reading the same', () => {
    expect(sameDeck(deck(), deck({ cards: read().cards.slice(0, 1) }))).toBe(false)
  })

  it('is not so for a field or a face written elsewhere', () => {
    expect(sameSheet(sheet(), sheet({ fields: ['Height'] }))).toBe(false)
    expect(sameSheet(sheet(), sheet({ faces: cut().faces.slice(0, 1) }))).toBe(false)
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
