/** What a gesture in a deck makes of the file, asked without a screen. */
import { describe, expect, it } from 'vitest'
import { CARD_HEAD, createCardRunEnd } from '@numen/ui'
import type { VaultDeck } from '../../cards'
import {
  addCard as addCardInternal,
  serializeBufferDeckToString,
  serializeBufferCardsToVaultCards,
  dropCard,
  createStencilChoices,
  deserializeBufferDeckFromString,
  deserializeVaultDeck,
  createDeckCards,
  replaceCardValue as replaceCardValueInternal,
  findStencilPath,
  removeCard,
  areDecksEqual,
  addSection,
  removeSection,
  renameSection,
  serializeBufferSectionsToVaultSections,
  type BufferDeck,
} from './deck'

type AddCardOptions = Parameters<typeof addCardInternal>[2]

const addCard = (
  deck: BufferDeck,
  title: string,
  stencilPath: string,
  values: AddCardOptions['values'],
  section: string | null = null,
  generateCardId?: AddCardOptions['generateCardId'],
) =>
  addCardInternal(deck, title, {
    stencilPath,
    values,
    section,
    ...(generateCardId ? { generateCardId } : {}),
  })

const replaceCardValue = (deck: BufferDeck, id: string, field: string, nth: number, text: string) =>
  replaceCardValueInternal(deck, id, { field, nth, text })

/** Identities counted out, so a test names the card it means. */
const createIds = () => {
  let at = 0
  return () => `c${(at += 1)}`
}

const readItem = (over: Partial<VaultDeck> = {}): VaultDeck => ({
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

const deck = (over: Partial<VaultDeck> = {}): BufferDeck =>
  deserializeVaultDeck(readItem(over), createIds())

describe('a deck as the window holds it', () => {
  it('knows every card by the mark the file carries for it', () => {
    expect(deck().cards.map((card) => card.id)).toStrictEqual([LLAMA, ALPACA])
  })

  it('mints an identity for a card carrying no mark, which the file cannot name', () => {
    const parsed = readItem()
    const card = deck({
      cards: [
        { ...parsed.cards[0]!, mark: '' },
        { ...parsed.cards[1]!, mark: '' },
      ],
    })
    expect(card.cards.map((card) => card.id)).toStrictEqual(['c1', 'c2'])
  })

  it('mints one for each of two cards carrying one mark, so both are drawn apart', () => {
    const parsed = readItem()
    const card = deck({ cards: [parsed.cards[0]!, { ...parsed.cards[1]!, mark: LLAMA }] })
    expect(card.cards.map((card) => card.id)).toStrictEqual(['c1', 'c2'])
  })

  it('gives every section an identity of its own, no file naming one', () => {
    const card = deck({ sections: [{ name: 'Roots', preamble: '' }] })
    expect(card.sections.map((section) => section.id)).toStrictEqual(['c1'])
  })

  it('remains each card under the section the file put it under', () => {
    const parsed = readItem()
    const card = deck({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: [{ ...parsed.cards[0]!, sectionIndex: 0 }, parsed.cards[1]!],
    })
    expect(card.cards.map((card) => card.section)).toStrictEqual([card.sections[0]?.id, null])
  })

  /* A section the reading does not hold is no section: the card stands before
     the first, drawn and counted, and is written back standing there. */
  it('remains a card before the first section where the reading holds no section it names', () => {
    const parsed = readItem()
    const card = deck({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: [{ ...parsed.cards[0]!, sectionIndex: 7 }, parsed.cards[1]!],
    })
    expect(card.cards.map((card) => card.section)).toStrictEqual([null, null])
    expect(serializeBufferCardsToVaultCards(card).map((card) => card.sectionIndex)).toStrictEqual([
      null,
      null,
    ])
  })

  it('keeps the preamble, the tail and each card’s preamble as the file had them', () => {
    const card = deck()
    expect(card.preamble).toBe('about the animals\n')
    expect(card.tail).toBe('\n')
    expect(card.cards[1]?.preamble).toBe('a note in the middle\n')
  })

  it('is the same string read out and written back', () => {
    const card = deck()
    expect(deserializeBufferDeckFromString(serializeBufferDeckToString(card))).toStrictEqual(card)
  })

  it('is a deck of no cards where nothing has been read', () => {
    expect(deserializeBufferDeckFromString('')).toStrictEqual({
      preamble: '',
      cards: [],
      sections: [],
      tail: '',
    })
  })

  it('passes the vault the cards without the identities it minted', () => {
    expect(serializeBufferCardsToVaultCards(deck())[0]).toStrictEqual({
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

  it('passes the vault each card under where its section remains among them', () => {
    const parsed = readItem()
    const card = deck({
      sections: [
        { name: 'Roots', preamble: '' },
        { name: 'Leaves', preamble: '' },
      ],
      cards: [{ ...parsed.cards[0]!, sectionIndex: 1 }, parsed.cards[1]!],
    })
    expect(serializeBufferCardsToVaultCards(card).map((card) => card.sectionIndex)).toStrictEqual([
      1,
      null,
    ])
  })

  it('passes the vault the sections without the identities it minted', () => {
    const card = deck({ sections: [{ name: 'Roots', preamble: 'about the roots\n' }] })
    expect(serializeBufferSectionsToVaultSections(card)).toStrictEqual([
      { name: 'Roots', preamble: 'about the roots\n' },
    ])
  })
})

describe('the cards as the grid draws them', () => {
  const OFFERS = [{ path: 'stencils/Animal.md', title: 'Animal', fields: ['Height', 'Life span'] }]

  /** One card, under the wikilink it wrote and the stencil that link reached. */
  const createCutDeck = (stencil: string, stencilPath: string): BufferDeck =>
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
    expect(createDeckCards(deck(), OFFERS)[0]).toStrictEqual({
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
    expect(JSON.stringify(createDeckCards(deck(), OFFERS))).not.toContain('a note in the middle')
  })

  it('draws a card the stencil its link reached, whatever stood in the brackets', () => {
    const content = ['cards/Animal', 'Animal|животное', 'Animal#Recognise', 'stencils/Animal.md']
    for (const one of content) {
      expect(createDeckCards(createCutDeck(one, 'stencils/Animal.md'), OFFERS)[0]?.stencil).toBe(
        'Animal',
      )
    }
  })

  it('draws a card whose link reached nothing under what the file wrote', () => {
    expect(createDeckCards(createCutDeck('Gone', ''), OFFERS)[0]?.stencil).toBe('Gone')
  })

  it('draws a card that wrote no brackets at all as cut by nothing', () => {
    expect(createDeckCards(createCutDeck('', ''), OFFERS)[0]?.stencil).toBeNull()
  })
})

describe('the stencils a card may be cut by', () => {
  it('are named by what each stencil is called', () => {
    expect(
      createStencilChoices([
        { path: 'stencils/Animal.md', title: 'Animal', fields: ['Height'] },
        { path: 'Term.md', title: 'Term', fields: ['Meaning'] },
      ]),
    ).toStrictEqual([
      { name: 'Animal', fields: ['Height'] },
      { name: 'Term', fields: ['Meaning'] },
    ])
  })

  it('name one stencil where two are called one name, and the first remains', () => {
    expect(
      createStencilChoices([
        { path: 'a/Animal.md', title: 'Animal', fields: ['Height'] },
        { path: 'b/Animal.md', title: 'Animal', fields: ['Weight'] },
      ]),
    ).toStrictEqual([{ name: 'Animal', fields: ['Height'] }])
  })

  it('leave out a stencil called nothing', () => {
    expect(createStencilChoices([{ path: 'Animal.md', title: '', fields: [] }])).toStrictEqual([])
  })

  it('are filed where the list returned, and nowhere for a name no stencil carries', () => {
    const offers = [{ path: 'stencils/Animal.md', title: 'Animal', fields: [] }]
    expect(findStencilPath(offers, 'Animal')).toBe('stencils/Animal.md')
    expect(findStencilPath(offers, 'Term')).toBe('')
  })
})

describe('a card added', () => {
  it('remains last, cut by the stencil it was requested for and carrying no mark', () => {
    const card = addCard(
      deck(),
      'Animal',
      'stencils/Animal.md',
      [{ field: 'Name', text: '' }],
      null,
      () => 'c9',
    )
    expect(card.cards.map((card) => card.id)).toStrictEqual([LLAMA, ALPACA, 'c9'])
    // A mark is written where the deck is made whole, on its way to the vault.
    expect(card.cards[2]).toStrictEqual({
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

  it('is placed under the section it was requested for, at the end of it', () => {
    const start = deck({
      sections: [
        { name: 'Roots', preamble: '' },
        { name: 'Leaves', preamble: '' },
      ],
    })
    const card = addCard(
      start,
      'Animal',
      'stencils/Animal.md',
      [],
      start.sections[1]!.id,
      () => 'c9',
    )
    expect(card.cards[2]?.section).toBe(start.sections[1]?.id)
  })

  it('remains before the first section where it was requested for none', () => {
    const start = deck({
      sections: [
        { name: 'Roots', preamble: '' },
        { name: 'Leaves', preamble: '' },
      ],
    })
    const card = addCard(start, 'Animal', 'stencils/Animal.md', [], null, () => 'c9')
    expect(card.cards.find((card) => card.id === 'c9')?.section).toBe(null)
  })

  it('leaves the preamble and the tail where they were', () => {
    const card = addCard(deck(), 'Animal', 'stencils/Animal.md', [], null, () => 'c9')
    expect(card.preamble).toBe('about the animals\n')
    expect(card.tail).toBe('\n')
  })

  it('names its stencil by the file, where the stencil is titled another way', () => {
    const card = addCard(deck(), 'Animal', 'stencils/creature stencil.md', [], null, () => 'c9')
    expect(card.cards[2]?.stencilLink).toBe('creature stencil')
  })

  it('names its stencil by no title, which a link resolves by nowhere', () => {
    const card = addCard(deck(), 'Animal', 'stencils/creature stencil.md', [], null, () => 'c9')
    expect(card.cards[2]?.stencilLink).not.toBe('Animal')
  })

  it('names it by the title where the vault filed the stencil nowhere', () => {
    const card = addCard(deck(), 'Animal', '', [], null, () => 'c9')
    expect(card.cards[2]?.stencilLink).toBe('Animal')
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
  const createSectionedDeck = (): BufferDeck => {
    const parsed = readItem()
    return deck({
      sections: [
        { name: 'Roots', preamble: '' },
        { name: 'Leaves', preamble: '' },
      ],
      cards: [parsed.cards[0]!, { ...parsed.cards[1]!, sectionIndex: 0 }],
    })
  }

  it('takes the section of the card it was let go before', () => {
    const card = createSectionedDeck()
    const moved = dropCard(card, LLAMA, ALPACA)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, card.sections[0]?.id],
      [ALPACA, card.sections[0]?.id],
    ])
  })

  it('is placed under the last section where it was let go on nothing', () => {
    const card = createSectionedDeck()
    expect(dropCard(card, LLAMA, null).cards.map((card) => card.section)).toStrictEqual([
      card.sections[0]?.id,
      card.sections[1]?.id,
    ])
  })

  it('lands at the head of a section it was let go on', () => {
    const card = createSectionedDeck()
    const roots = card.sections[0]?.id ?? ''
    const moved = dropCard(card, LLAMA, roots)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, roots],
      [ALPACA, roots],
    ])
  })

  it('lands at the head of a section holding no card', () => {
    const card = createSectionedDeck()
    const leaves = card.sections[1]?.id ?? ''
    const moved = dropCard(card, LLAMA, leaves)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, card.sections[0]?.id],
      [LLAMA, leaves],
    ])
  })

  it('moves nothing where it was let go on nothing the deck holds', () => {
    const card = createSectionedDeck()
    expect(dropCard(card, LLAMA, 'nowhere')).toStrictEqual(card)
  })

  it('remains last under the section it was let go past the end of', () => {
    const card = createSectionedDeck()
    const roots = card.sections[0]?.id ?? ''
    const moved = dropCard(card, LLAMA, createCardRunEnd(roots))
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, roots],
      [LLAMA, roots],
    ])
  })

  it('remains last under no section where it was let go past those before the first', () => {
    const card = createSectionedDeck()
    const moved = dropCard(card, ALPACA, createCardRunEnd(CARD_HEAD))
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, null],
      [ALPACA, null],
    ])
  })

  // A card naming a section the deck does not hold is drawn before the first
  // heading, so that is where it counts from when another lands past it.
  it('remains past a card whose section the deck has lost, which is placed under none', () => {
    const card = createSectionedDeck()
    const missing = {
      ...card.cards[0]!,
      section: 'gone',
    }
    const deckLost = { ...card, cards: [missing, card.cards[1]!] }

    const moved = dropCard(deckLost, ALPACA, createCardRunEnd(CARD_HEAD))

    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [LLAMA, 'gone'],
      [ALPACA, null],
    ])
  })

  it('leaves the deck as it was where the card already remains last under that heading', () => {
    const card = createSectionedDeck()
    expect(dropCard(card, ALPACA, createCardRunEnd(card.sections[0]?.id ?? ''))).toStrictEqual(card)
  })

  it('remains first and under no section where it was let go at the head of the deck', () => {
    const card = createSectionedDeck()
    const moved = dropCard(card, ALPACA, CARD_HEAD)
    expect(moved.cards.map((card) => [card.id, card.section])).toStrictEqual([
      [ALPACA, null],
      [LLAMA, null],
    ])
  })
})

describe('a section of a deck', () => {
  const createSectionedDeck = (): BufferDeck =>
    deck({
      sections: [
        { name: 'Roots', preamble: '' },
        { name: 'Leaves', preamble: '' },
      ],
    })

  it('is made at the end of the deck, holding no card', () => {
    const card = addSection(createSectionedDeck(), 'Shoots', () => 'c9')
    expect(card.sections[2]).toStrictEqual({ id: 'c9', name: 'Shoots', preamble: '' })
    expect(card.cards.map((card) => card.section)).toStrictEqual([null, null])
  })

  it('takes the name it was given, and no other section takes it', () => {
    const card = createSectionedDeck()
    const named = renameSection(card, card.sections[0]?.id ?? '', 'Roots and shoots')
    expect(named.sections.map((section) => section.name)).toStrictEqual([
      'Roots and shoots',
      'Leaves',
    ])
  })

  it('takes a name the section alongside it carries, two being free to share one', () => {
    const card = createSectionedDeck()
    const named = renameSection(card, card.sections[0]?.id ?? '', 'Leaves')
    expect(named.sections.map((section) => section.name)).toStrictEqual(['Leaves', 'Leaves'])
  })

  /* Taking a section away takes away its heading and nothing else: its cards
     stay where they stand, under whatever heading is above them now. */
  it('leaves its cards under the section above it when it goes', () => {
    const parsed = readItem()
    const card = deck({
      sections: [
        { name: 'Roots', preamble: '' },
        { name: 'Leaves', preamble: '' },
      ],
      cards: [
        { ...parsed.cards[0]!, sectionIndex: 0 },
        { ...parsed.cards[1]!, sectionIndex: 1 },
      ],
    })
    const removed = removeSection(card, card.sections[1]?.id ?? '')
    expect(removed.sections.map((section) => section.name)).toStrictEqual(['Roots'])
    expect(removed.cards.map((card) => card.section)).toStrictEqual([
      card.sections[0]?.id,
      card.sections[0]?.id,
    ])
  })

  /* A section is a name, so taking it away takes away a name: the text it stood
     on stays in the deck, standing after the text above it. */
  it('leaves the text it stood on with the section above it when it goes', () => {
    const card = deck({
      sections: [
        { name: 'Roots', preamble: 'about the roots\n' },
        { name: 'Leaves', preamble: 'about the leaves\n' },
      ],
    })
    const removed = removeSection(card, card.sections[1]?.id ?? '')
    expect(removed.sections.map((section) => section.preamble)).toStrictEqual([
      'about the roots\n\nabout the leaves\n',
    ])
  })

  it('leaves it with the deck’s own text where no section remains above it', () => {
    const card = deck({ sections: [{ name: 'Roots', preamble: 'about the roots\n' }] })
    const removed = removeSection(card, card.sections[0]?.id ?? '')
    expect(removed.preamble).toBe('about the animals\n\nabout the roots\n')
  })

  it('leaves the text above it alone where it stood on none of its own', () => {
    const card = deck({
      sections: [
        { name: 'Roots', preamble: 'about the roots\n' },
        { name: 'Leaves', preamble: '' },
      ],
    })
    const removed = removeSection(card, card.sections[1]?.id ?? '')
    expect(removed.sections.map((section) => section.preamble)).toStrictEqual(['about the roots\n'])
  })

  it('leaves its cards under no section when the first of them goes', () => {
    const parsed = readItem()
    const card = deck({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: [{ ...parsed.cards[0]!, sectionIndex: 0 }, parsed.cards[1]!],
    })
    const removed = removeSection(card, card.sections[0]?.id ?? '')
    expect(removed.sections).toStrictEqual([])
    expect(removed.cards.map((card) => card.section)).toStrictEqual([null, null])
  })
})

describe('a value written into a card', () => {
  it('remains where the field already was', () => {
    const card = replaceCardValue(deck(), LLAMA, 'Height', 1, 'about 46"')
    expect(card.cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 46"' },
      { field: 'Life span', text: 'about 20 years' },
    ])
  })

  it('is the one of two under a field that was typed in, and the other remains', () => {
    const twice = deck()
    const targetCard = twice.cards[0]
    if (!targetCard) throw new Error('card not found')
    const updatedDeck = replaceCardValue(
      {
        ...twice,
        cards: [{ ...targetCard, values: [...targetCard.values, { field: 'Height', text: 'about 46"' }] }],
      },
      LLAMA,
      'Height',
      2,
      'about 47"',
    )
    expect(updatedDeck.cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
      { field: 'Life span', text: 'about 20 years' },
      { field: 'Height', text: 'about 47"' },
    ])
  })

  it('is written after the rest where the card had no such field', () => {
    const card = replaceCardValue(deck(), LLAMA, 'Weight', 1, '130 kg')
    expect(card.cards[0]?.values.map((value) => value.field)).toStrictEqual([
      'Name',
      'Height',
      'Life span',
      'Weight',
    ])
  })

  it('keeps the heading of a field the card has, emptied', () => {
    const card = replaceCardValue(deck(), LLAMA, 'Height', 1, '')
    expect(card.cards[0]?.values[1]).toStrictEqual({ field: 'Height', text: '' })
  })

  it('writes no heading for a field the card does not have and nothing was typed into', () => {
    expect(replaceCardValue(deck(), ALPACA, 'Height', 1, '').cards[1]?.values).toStrictEqual([])
  })

  it('leaves every other card as it was', () => {
    expect(replaceCardValue(deck(), LLAMA, 'Height', 1, 'taller').cards[1]).toStrictEqual(
      deck().cards[1],
    )
  })
})

describe('whether two evaluations of a file read the same', () => {
  it('is so for one file read twice, whatever identities each reading minted', () => {
    expect(areDecksEqual(deck(), deck())).toBe(true)
  })

  it('is not so for a card whose text was written elsewhere', () => {
    const other = readItem()
    const wrote = {
      ...other,
      cards: [{ ...other.cards[0]!, values: [{ field: 'Height', text: 'about 6ft' }] }],
    }

    expect(areDecksEqual(deck(), deserializeVaultDeck(wrote, createIds()))).toBe(false)
  })

  // The heading is read back from the first field wherever the deck is written,
  // so a save carrying a value the window typed comes back under a heading the
  // window never held. That is the vault keeping its own word, not somebody
  // writing this deck elsewhere, and what a person is typing into stands.
  it('is so where only the heading a write read back is different', () => {
    const other = readItem()
    const wrote = {
      ...other,
      cards: [{ ...other.cards[0]!, heading: 'about 45"' }, ...other.cards.slice(1)],
    }

    expect(areDecksEqual(deck(), deserializeVaultDeck(wrote, createIds()))).toBe(true)
  })

  it('is not so for prose around the cards written elsewhere', () => {
    expect(areDecksEqual(deck(), deck({ preamble: 'about them\n' }))).toBe(false)
    expect(areDecksEqual(deck(), deck({ tail: '\n\n' }))).toBe(false)
  })

  it('is not so for a card taken out, the ones left over reading the same', () => {
    expect(areDecksEqual(deck(), deck({ cards: readItem().cards.slice(0, 1) }))).toBe(false)
  })
})

describe('cards and sections standing among three headings', () => {
  /** Cards a (before the first heading), b (Roots) and c (Leaves), then Shoots holding none. */
  const createThreeHeadingDeck = (): BufferDeck => {
    const baseCard = readItem().cards[0]!
    return deck({
      sections: [
        { name: 'Roots', preamble: 'about the roots\n' },
        { name: 'Leaves', preamble: 'about the leaves\n' },
        { name: 'Shoots', preamble: 'about the shoots\n' },
      ],
      cards: [
        { ...baseCard, mark: 'ma', sectionIndex: null },
        { ...baseCard, mark: 'mb', sectionIndex: 0 },
        { ...baseCard, mark: 'mc', sectionIndex: 1 },
      ],
    })
  }

  const place = (card: BufferDeck) => card.cards.map((card) => [card.id, card.section])

  describe('a card added', () => {
    it('remains after the cards before the first heading and ahead of the headed ones', () => {
      const card = createThreeHeadingDeck()
      const added = addCard(card, 'Animal', '', [], null, () => 'new')
      expect(place(added)).toStrictEqual([
        ['ma', null],
        ['new', null],
        ['mb', card.sections[0]?.id],
        ['mc', card.sections[1]?.id],
      ])
    })

    it('remains after the cards of the section above and ahead of the section below', () => {
      const card = createThreeHeadingDeck()
      const roots = card.sections[0]?.id ?? ''
      const added = addCard(card, 'Animal', '', [], roots, () => 'new')
      expect(place(added)).toStrictEqual([
        ['ma', null],
        ['mb', roots],
        ['new', roots],
        ['mc', card.sections[1]?.id],
      ])
    })

    it('remains last under the last section that holds cards, when the section is below it', () => {
      const card = createThreeHeadingDeck()
      const leaves = card.sections[1]?.id ?? ''
      const added = addCard(card, 'Animal', '', [], leaves, () => 'new')
      expect(place(added).at(-1)).toStrictEqual(['new', leaves])
    })

    it('remains before the first heading where the section it was requested for is not in the deck', () => {
      const currentDeck = createThreeHeadingDeck()
      const added = addCard(currentDeck, 'Animal', '', [], 'ghost', () => 'new')
      expect(place(added)).toStrictEqual([
        ['ma', null],
        ['new', null],
        ['mb', currentDeck.sections[0]?.id],
        ['mc', currentDeck.sections[1]?.id],
      ])
    })
  })

  describe('a card dragged', () => {
    it('is nothing for an identity the deck does not hold', () => {
      const currentDeck = createThreeHeadingDeck()
      expect(dropCard(currentDeck, 'ghost', null)).toBe(currentDeck)
    })

    it('lands before a card that is not the first, leaving the cards ahead of it in place', () => {
      const currentDeck = createThreeHeadingDeck()
      const moved = dropCard(currentDeck, 'ma', 'mc')
      expect(place(moved)).toStrictEqual([
        ['mb', currentDeck.sections[0]?.id],
        ['ma', currentDeck.sections[1]?.id],
        ['mc', currentDeck.sections[1]?.id],
      ])
    })

    it('is placed under the last section of the deck where it was let go on nothing', () => {
      const currentDeck = createThreeHeadingDeck()
      const moved = dropCard(currentDeck, 'ma', null)
      expect(place(moved).at(-1)).toStrictEqual(['ma', currentDeck.sections[2]?.id])
    })

    it('remains ahead of the cards of later sections where the section it was let go past holds none', () => {
      const currentDeck = createThreeHeadingDeck()
      const roots = currentDeck.sections[0]?.id ?? ''
      const moved = dropCard(removeCard(currentDeck, 'mb'), 'ma', createCardRunEnd(roots))
      expect(place(moved)).toStrictEqual([
        ['ma', roots],
        ['mc', currentDeck.sections[1]?.id],
      ])
    })

    it('remains after the last card of the section it was let go past, ahead of later sections', () => {
      const currentDeck = createThreeHeadingDeck()
      const roots = currentDeck.sections[0]?.id ?? ''
      const moved = dropCard(currentDeck, 'ma', createCardRunEnd(roots))
      expect(place(moved)).toStrictEqual([
        ['mb', roots],
        ['ma', roots],
        ['mc', currentDeck.sections[1]?.id],
      ])
    })

    it('moves nothing where it was let go past a section the deck does not hold', () => {
      const currentDeck = createThreeHeadingDeck()
      expect(dropCard(currentDeck, 'ma', createCardRunEnd('ghost'))).toStrictEqual(currentDeck)
    })
  })

  describe('a section removed', () => {
    it('is nothing for an identity the deck does not hold', () => {
      const currentDeck = createThreeHeadingDeck()
      expect(removeSection(currentDeck, 'ghost')).toBe(currentDeck)
    })

    it('leaves the text of the sections other than the one above it as it was', () => {
      const currentDeck = createThreeHeadingDeck()
      const remainingDeck = removeSection(currentDeck, currentDeck.sections[2]?.id ?? '')
      expect(remainingDeck.sections.map((section) => section.preamble)).toStrictEqual([
        'about the roots\n',
        'about the leaves\n\nabout the shoots\n',
      ])
    })

    it('leaves the text of the sections below it as it was where it is the first', () => {
      const currentDeck = createThreeHeadingDeck()
      const remainingDeck = removeSection(currentDeck, currentDeck.sections[0]?.id ?? '')
      expect(remainingDeck.preamble).toBe('about the animals\n\nabout the roots\n')
      expect(remainingDeck.sections.map((section) => section.preamble)).toStrictEqual([
        'about the leaves\n',
        'about the shoots\n',
      ])
    })

    it('leaves the cards of other sections under the sections they stood under', () => {
      const currentDeck = createThreeHeadingDeck()
      const remainingDeck = removeSection(currentDeck, currentDeck.sections[2]?.id ?? '')
      expect(place(remainingDeck)).toStrictEqual(place(currentDeck))
    })
  })
})
