/** What a gesture in a deck makes of the file, asked without a screen. */
import { describe, expect, it } from 'vitest'
import { CARD_HEAD, cardEndOf } from '@numen/ui'
import type { VaultDeck } from '../../entities/deck/cards'
import {
  addCard,
  deckBodyOf,
  cardsOf,
  dropCard,
  stencilsOf,
  deckIn,
  deckOf,
  drawnOf,
  fillCard,
  pathOfCut,
  removeCard,
  sameDeck,
  addSection,
  removeSection,
  renameSection,
  sectionsOf,
  type BufferDeck,
} from './deck'

/** Identities counted out, so a test names the card it means. */
const minting = () => {
  let at = 0
  return () => `c${(at += 1)}`
}

const read = (over: Partial<VaultDeck> = {}): VaultDeck => ({
  path: 'Animals.md',
  title: 'Animals',
  preamble: 'about the animals\n',
  cards: [
    {
      mark: 'k7m2xq9fzp',
      sectionIndex: null,
      heading: 'Llama',
      stencilLink: 'Animal',
      stencilPath: 'stencils/Animal.md',
      preamble: '',
      values: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: 'about 45"' },
        { field: 'Life span', text: 'about 20 years' },
      ],
    },
    {
      mark: '3n8vr4tqch',
      sectionIndex: null,
      heading: 'Alpaca',
      stencilLink: 'Animal',
      stencilPath: 'stencils/Animal.md',
      preamble: 'a note in the middle\n',
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

const deck = (over: Partial<VaultDeck> = {}): BufferDeck => deckOf(read(over), minting())

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
    const held = deck({ sections: [{ name: 'Roots', preamble: '' }] })
    expect(held.sections.map((section) => section.id)).toStrictEqual(['c1'])
  })

  it('stands each card under the section the file put it under', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: [{ ...bare.cards[0]!, sectionIndex:0 }, bare.cards[1]!],
    })
    expect(held.cards.map((card) => card.section)).toStrictEqual([held.sections[0]?.id, null])
  })

  /* A section the reading does not hold is no section: the card stands before
     the first, drawn and counted, and is written back standing there. */
  it('stands a card before the first section where the reading holds no section it names', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: [{ ...bare.cards[0]!, sectionIndex:7 }, bare.cards[1]!],
    })
    expect(held.cards.map((card) => card.section)).toStrictEqual([null, null])
    expect(cardsOf(held).map((card) => card.sectionIndex)).toStrictEqual([null, null])
  })

  it('keeps the preamble, the tail and each card’s preamble as the file had them', () => {
    const held = deck()
    expect(held.preamble).toBe('about the animals\n')
    expect(held.tail).toBe('\n')
    expect(held.cards[1]?.preamble).toBe('a note in the middle\n')
  })

  it('is the same string read out and written back', () => {
    const held = deck()
    expect(deckIn(deckBodyOf(held))).toStrictEqual(held)
  })

  it('is a deck of no cards where nothing has been read', () => {
    expect(deckIn('')).toStrictEqual({ preamble: '', cards: [], sections: [], tail: '' })
  })

  it('hands the vault the cards without the identities it minted', () => {
    expect(cardsOf(deck())[0]).toStrictEqual({
      mark: LLAMA,
      sectionIndex: null,
      heading: 'Llama',
      stencilLink: 'Animal',
      stencilPath: 'stencils/Animal.md',
      preamble: '',
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
      sections: [{ name: 'Roots', preamble: '' }, { name: 'Leaves', preamble: '' }],
      cards: [{ ...bare.cards[0]!, sectionIndex:1 }, bare.cards[1]!],
    })
    expect(cardsOf(held).map((card) => card.sectionIndex)).toStrictEqual([1, null])
  })

  it('hands the vault the sections without the identities it minted', () => {
    const held = deck({ sections: [{ name: 'Roots', preamble: 'about the roots\n' }] })
    expect(sectionsOf(held)).toStrictEqual([{ name: 'Roots', preamble: 'about the roots\n' }])
  })
})

describe('the cards as the grid draws them', () => {
  const OFFERS = [
    { path: 'stencils/Animal.md', title: 'Animal', fields: ['Height', 'Life span'] },
  ]

  /** One card, under the wikilink it wrote and the stencil that link reached. */
  const cutBy = (stencil: string, stencilPath: string): BufferDeck =>
    deck({
      cards: [
        {
          mark: LLAMA,
          sectionIndex: null,
          heading: 'Llama',
          stencilLink: stencil,
          stencilPath,
          preamble: '',
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

  it('says nothing of the preamble, which nothing lays out', () => {
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
    const held = addCard(
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
      stencilLink: 'Animal',
      stencilPath: 'stencils/Animal.md',
      preamble: '',
      values: [{ field: 'Name', text: '' }],
    })
  })

  it('stands under the section it was asked for, at the end of it', () => {
    const start = deck({ sections: [{ name: 'Roots', preamble: '' }, { name: 'Leaves', preamble: '' }] })
    const held = addCard(start, 'Animal', 'stencils/Animal.md', [], start.sections[1]!.id, () => 'c9')
    expect(held.cards[2]?.section).toBe(start.sections[1]?.id)
  })

  it('stands before the first section where it was asked for none', () => {
    const start = deck({ sections: [{ name: 'Roots', preamble: '' }, { name: 'Leaves', preamble: '' }] })
    const held = addCard(start, 'Animal', 'stencils/Animal.md', [], null, () => 'c9')
    expect(held.cards.find((card) => card.id === 'c9')?.section).toBe(null)
  })

  it('leaves the preamble and the tail where they were', () => {
    const held = addCard(deck(), 'Animal', 'stencils/Animal.md', [], null, () => 'c9')
    expect(held.preamble).toBe('about the animals\n')
    expect(held.tail).toBe('\n')
  })

  it('names its stencil by the file, where the stencil is titled another way', () => {
    const held = addCard(deck(), 'Animal', 'stencils/creature stencil.md', [], null, () => 'c9')
    expect(held.cards[2]?.stencilLink).toBe('creature stencil')
  })

  it('names its stencil by no title, which a link resolves by nowhere', () => {
    const held = addCard(deck(), 'Animal', 'stencils/creature stencil.md', [], null, () => 'c9')
    expect(held.cards[2]?.stencilLink).not.toBe('Animal')
  })

  it('names it by the title where the vault filed the stencil nowhere', () => {
    const held = addCard(deck(), 'Animal', '', [], null, () => 'c9')
    expect(held.cards[2]?.stencilLink).toBe('Animal')
  })
})

describe('a card taken out and dragged', () => {
  it('goes, and the rest stay in the order they were in', () => {
    expect(removeCard(deck(), LLAMA).cards.map((card) => card.id)).toStrictEqual([ALPACA])
  })

  it('is nothing for an identity the deck does not hold', () => {
    expect(removeCard(deck(), 'c9').cards).toHaveLength(2)
  })

  it('lands before the card it was let go on', () => {
    expect(dropCard(deck(), ALPACA, LLAMA).cards.map((card) => card.id)).toStrictEqual([
      ALPACA,
      LLAMA,
    ])
  })

  it('lands last where it was let go on nothing', () => {
    expect(dropCard(deck(), LLAMA, null).cards.map((card) => card.id)).toStrictEqual([
      ALPACA,
      LLAMA,
    ])
  })
})

describe('a card dragged among the sections', () => {
  /** Two sections, the first card in the first of them and the second in neither. */
  const sectioned = (): BufferDeck => {
    const bare = read()
    return deck({
      sections: [{ name: 'Roots', preamble: '' }, { name: 'Leaves', preamble: '' }],
      cards: [bare.cards[0]!, { ...bare.cards[1]!, sectionIndex:0 }],
    })
  }

  it('takes the section of the card it was let go before', () => {
    const held = sectioned()
    const moved = dropCard(held, LLAMA, ALPACA)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, held.sections[0]?.id],
      [ALPACA, held.sections[0]?.id],
    ])
  })

  it('stands under the last section where it was let go on nothing', () => {
    const held = sectioned()
    expect(dropCard(held, LLAMA, null).cards.map((card) => card.section)).toStrictEqual([
      held.sections[0]?.id,
      held.sections[1]?.id,
    ])
  })

  it('lands at the head of a section it was let go on', () => {
    const held = sectioned()
    const roots = held.sections[0]?.id ?? ''
    const moved = dropCard(held, LLAMA, roots)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, roots],
      [ALPACA, roots],
    ])
  })

  it('lands at the head of a section holding no card', () => {
    const held = sectioned()
    const leaves = held.sections[1]?.id ?? ''
    const moved = dropCard(held, LLAMA, leaves)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, held.sections[0]?.id],
      [LLAMA, leaves],
    ])
  })

  it('moves nothing where it was let go on nothing the deck holds', () => {
    const held = sectioned()
    expect(dropCard(held, LLAMA, 'nowhere')).toStrictEqual(held)
  })

  it('stands last under the section it was let go past the end of', () => {
    const held = sectioned()
    const roots = held.sections[0]?.id ?? ''
    const moved = dropCard(held, LLAMA, cardEndOf(roots))
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, roots],
      [LLAMA, roots],
    ])
  })

  it('stands last under no section where it was let go past those before the first', () => {
    const held = sectioned()
    const moved = dropCard(held, ALPACA, cardEndOf(CARD_HEAD))
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

    const moved = dropCard(deckLost, ALPACA, cardEndOf(CARD_HEAD))

    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, 'gone'],
      [ALPACA, null],
    ])
  })

  it('leaves the deck as it was where the card already stands last under that heading', () => {
    const held = sectioned()
    expect(dropCard(held, ALPACA, cardEndOf(held.sections[0]?.id ?? ''))).toStrictEqual(held)
  })

  it('stands first and under no section where it was let go at the head of the deck', () => {
    const held = sectioned()
    const moved = dropCard(held, ALPACA, CARD_HEAD)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, null],
      [LLAMA, null],
    ])
  })
})

describe('a section of a deck', () => {
  const sectioned = (): BufferDeck =>
    deck({ sections: [{ name: 'Roots', preamble: '' }, { name: 'Leaves', preamble: '' }] })

  it('is made at the end of the deck, holding no card', () => {
    const held = addSection(sectioned(), 'Shoots', () => 'c9')
    expect(held.sections[2]).toStrictEqual({ id: 'c9', name: 'Shoots', preamble: '' })
    expect(held.cards.map((card) => card.section)).toStrictEqual([null, null])
  })

  it('takes the name it was given, and no other section takes it', () => {
    const held = sectioned()
    const named = renameSection(held, held.sections[0]?.id ?? '', 'Roots and shoots')
    expect(named.sections.map((section) => section.name)).toStrictEqual([
      'Roots and shoots',
      'Leaves',
    ])
  })

  it('takes a name the section beside it carries, two being free to share one', () => {
    const held = sectioned()
    const named = renameSection(held, held.sections[0]?.id ?? '', 'Leaves')
    expect(named.sections.map((section) => section.name)).toStrictEqual(['Leaves', 'Leaves'])
  })

  /* Taking a section away takes away its heading and nothing else: its cards
     stay where they stand, under whatever heading is above them now. */
  it('leaves its cards under the section above it when it goes', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', preamble: '' }, { name: 'Leaves', preamble: '' }],
      cards: [{ ...bare.cards[0]!, sectionIndex: 0 }, { ...bare.cards[1]!, sectionIndex: 1 }],
    })
    const gone = removeSection(held, held.sections[1]?.id ?? '')
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
        { name: 'Roots', preamble: 'about the roots\n' },
        { name: 'Leaves', preamble: 'about the leaves\n' },
      ],
    })
    const gone = removeSection(held, held.sections[1]?.id ?? '')
    expect(gone.sections.map((section) => section.preamble)).toStrictEqual([
      'about the roots\n\nabout the leaves\n',
    ])
  })

  it('leaves it with the deck’s own text where no section stands above it', () => {
    const held = deck({ sections: [{ name: 'Roots', preamble: 'about the roots\n' }] })
    const gone = removeSection(held, held.sections[0]?.id ?? '')
    expect(gone.preamble).toBe('about the animals\n\nabout the roots\n')
  })

  it('leaves the text above it alone where it stood on none of its own', () => {
    const held = deck({
      sections: [{ name: 'Roots', preamble: 'about the roots\n' }, { name: 'Leaves', preamble: '' }],
    })
    const gone = removeSection(held, held.sections[1]?.id ?? '')
    expect(gone.sections.map((section) => section.preamble)).toStrictEqual(['about the roots\n'])
  })

  it('leaves its cards under no section when the first of them goes', () => {
    const bare = read()
    const held = deck({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: [{ ...bare.cards[0]!, sectionIndex: 0 }, bare.cards[1]!],
    })
    const gone = removeSection(held, held.sections[0]?.id ?? '')
    expect(gone.sections).toStrictEqual([])
    expect(gone.cards.map((card) => card.section)).toStrictEqual([null, null])
  })
})

describe('a value written into a card', () => {
  it('stands where the field already was', () => {
    const held = fillCard(deck(), LLAMA, 'Height', 1, 'about 46"')
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
    const held = fillCard(
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
    const held = fillCard(deck(), LLAMA, 'Weight', 1, '130 kg')
    expect(held.cards[0]?.values.map((value) => value.field)).toStrictEqual([
      'Name',
      'Height',
      'Life span',
      'Weight',
    ])
  })

  it('keeps the heading of a field the card has, emptied', () => {
    const held = fillCard(deck(), LLAMA, 'Height', 1, '')
    expect(held.cards[0]?.values[1]).toStrictEqual({ field: 'Height', text: '' })
  })

  it('writes no heading for a field the card does not have and nothing was typed into', () => {
    expect(fillCard(deck(), ALPACA, 'Height', 1, '').cards[1]?.values).toStrictEqual([])
  })

  it('leaves every other card as it was', () => {
    expect(fillCard(deck(), LLAMA, 'Height', 1, 'taller').cards[1]).toStrictEqual(
      deck().cards[1],
    )
  })
})

describe('whether two readings of a file read the same', () => {
  it('is so for one file read twice, whatever identities each reading minted', () => {
    expect(sameDeck(deck(), deck())).toBe(true)
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
})
