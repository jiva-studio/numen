/**
 * A reading of a deck file meets the deck on screen: headings come back from the
 * file, identities stay with the cards and sections they were drawn by, and a
 * stencil listing is compared stencil by stencil.
 */
import { describe, expect, it } from 'vitest'
import type { StencilSummary } from '../../cards'
import type { BufferCard, BufferDeck, BufferSection } from '../types'
import { applyReadHeadings, applyScreenIds, areDecksEqual, areOffersEqual } from './view'

const card = (overrides: Partial<BufferCard> = {}): BufferCard => ({
  id: 'screen-card',
  mark: '',
  section: null,
  heading: 'Llama',
  stencilLink: 'Animal',
  stencilPath: 'stencils/Animal.md',
  preamble: '',
  values: [{ field: 'Name', text: 'Llama' }],
  ...overrides,
})

const section = (overrides: Partial<BufferSection> = {}): BufferSection => ({
  id: 'screen-section',
  name: 'Mammals',
  preamble: '',
  ...overrides,
})

const deck = (overrides: Partial<BufferDeck> = {}): BufferDeck => ({
  preamble: '',
  cards: [],
  sections: [],
  tail: '',
  ...overrides,
})

/** The card just written by the window, and the same card read back with its new mark. */
const data = (overrides: Partial<BufferCard> = {}) => ({
  screen: deck({ cards: [card()] }),
  read: deck({ cards: [card({ id: 'k7m2xq9fzp', mark: 'k7m2xq9fzp', ...overrides })] }),
})

describe('whether two decks read the same', () => {
  it('is not so for a section renamed, every card reading the same', () => {
    const one = deck({ sections: [section()] })
    const other = deck({ sections: [section({ name: 'Birds' })] })

    expect(areDecksEqual(one, other)).toBe(false)
  })
})

describe('the deck on screen under the headings the file carries', () => {
  it('is the deck on screen itself where every heading already matches', () => {
    const screen = deck({ cards: [card()] })

    expect(applyReadHeadings(screen, deck({ cards: [card({ id: 'other' })] }))).toBe(screen)
  })

  it('takes each heading from the card read at the same place', () => {
    const screen = deck({ cards: [card(), card({ id: 'second', heading: 'Alpaca' })] })
    const readItem = deck({ cards: [card({ heading: 'about 45"' }), card({ heading: 'Alpaca' })] })

    expect(applyReadHeadings(screen, readItem).cards.map((one) => one.heading)).toStrictEqual([
      'about 45"',
      'Alpaca',
    ])
  })

  it('gives a card the file does not carry yet an empty heading', () => {
    const screen = deck({ cards: [card(), card({ id: 'second' })] })
    const readItem = deck({ cards: [card()] })

    expect(applyReadHeadings(screen, readItem).cards.map((one) => one.heading)).toStrictEqual([
      'Llama',
      '',
    ])
  })
})

describe('a reading under the identities the window drew it by', () => {
  it('keeps the identity of a section read back under the same name and prose', () => {
    const screen = deck({ sections: [section()] })
    const readItem = deck({ sections: [section({ id: 'read-section' })] })

    expect(applyScreenIds(screen, readItem).sections.map((one) => one.id)).toStrictEqual([
      'screen-section',
    ])
  })

  it('takes the identity read for a section renamed', () => {
    const screen = deck({ sections: [section()] })
    const readItem = deck({ sections: [section({ id: 'read-section', name: 'Birds' })] })

    expect(applyScreenIds(screen, readItem).sections.map((one) => one.id)).toStrictEqual([
      'read-section',
    ])
  })

  it('takes the identity read for a section whose prose changed', () => {
    const screen = deck({ sections: [section()] })
    const readItem = deck({ sections: [section({ id: 'read-section', preamble: 'warm blood\n' })] })

    expect(applyScreenIds(screen, readItem).sections.map((one) => one.id)).toStrictEqual([
      'read-section',
    ])
  })

  it('takes every identity read where a section was added', () => {
    const screen = deck({ sections: [section()] })
    const readItem = deck({
      sections: [section({ id: 'read-1' }), section({ id: 'read-2', name: 'Birds' })],
    })

    expect(applyScreenIds(screen, readItem).sections.map((one) => one.id)).toStrictEqual([
      'read-1',
      'read-2',
    ])
  })

  it('files each card under the section it was read under, by the identity kept for it', () => {
    const screen = deck({ sections: [section({ id: 's1' }), section({ id: 's2', name: 'Birds' })] })
    const readItem = deck({
      sections: [section({ id: 'r1' }), section({ id: 'r2', name: 'Birds' })],
      cards: [card({ section: 'r2' }), card({ id: 'loose', section: null })],
    })

    expect(applyScreenIds(screen, readItem).cards.map((one) => one.section)).toStrictEqual(['s2', null])
  })

  it('files a card under no section where the section it names was not read', () => {
    const readItem = deck({ sections: [section({ id: 'r1' })], cards: [card({ section: 'ghost' })] })

    expect(applyScreenIds(deck(), readItem).cards[0]?.section).toBeNull()
  })

  it('keeps the identity of a card the window wrote, read back under its new mark', () => {
    const { screen, read } = data()

    expect(applyScreenIds(screen, read).cards[0]?.id).toBe('screen-card')
  })

  it('keeps the identity of a card the window wrote into a section', () => {
    const screen = deck({ sections: [section()], cards: [card({ section: 'screen-section' })] })
    const readItem = deck({
      sections: [section({ id: 'r1' })],
      cards: [card({ id: 'k7m2xq9fzp', mark: 'k7m2xq9fzp', section: 'r1' })],
    })

    expect(applyScreenIds(screen, readItem).cards[0]).toMatchObject({
      id: 'screen-card',
      section: 'screen-section',
    })
  })

  it.each([
    ['its stencil', { stencilLink: 'Bird' }],
    ['its prose', { preamble: 'a note\n' }],
    ['its values', { values: [{ field: 'Name', text: 'Alpaca' }] }],
  ])('takes the identity read for a card whose %s changed', (_what, override) => {
    const { screen, read } = data(override)

    expect(applyScreenIds(screen, read).cards[0]?.id).toBe('k7m2xq9fzp')
  })

  it('takes the identity read for a card that moved into a section', () => {
    const screen = deck({ sections: [section(), section({ id: 's2', name: 'Birds' })] })
    const readItem = deck({
      sections: [section({ id: 'r1' }), section({ id: 'r2', name: 'Birds' })],
      cards: [card({ id: 'k7m2xq9fzp', mark: 'k7m2xq9fzp', section: 'r2' })],
    })

    expect(applyScreenIds({ ...screen, cards: [card({ section: null })] }, readItem).cards[0]?.id).toBe(
      'k7m2xq9fzp',
    )
  })

  it('takes the identity read for a card where the file holds another count of cards', () => {
    const { screen, read } = data()
    const more = { ...read, cards: [...read.cards, card({ id: 'second', mark: 'second' })] }

    expect(applyScreenIds(screen, more).cards[0]?.id).toBe('k7m2xq9fzp')
  })

  it('takes the identity read for a card the screen already knew by a mark', () => {
    const screen = deck({ cards: [card({ id: 'old-mark', mark: 'old-mark' })] })
    const { read } = data()

    expect(applyScreenIds(screen, read).cards[0]?.id).toBe('k7m2xq9fzp')
  })

  it('takes the identity read for a card the file still carries no mark for', () => {
    const { screen } = data()
    const readItem = deck({ cards: [card({ id: 'minted' })] })

    expect(applyScreenIds(screen, readItem).cards[0]?.id).toBe('minted')
  })
})

describe('whether two stencil listings are the same', () => {
  const offer = (overrides: Partial<StencilSummary> = {}): StencilSummary => ({
    path: 'stencils/Animal.md',
    title: 'Animal',
    fields: ['Name', 'Height'],
    ...overrides,
  })

  it('is so for the same stencils in the same order', () => {
    expect(
      areOffersEqual([offer(), offer({ path: 'b.md' })], [offer(), offer({ path: 'b.md' })]),
    ).toBe(true)
  })

  it.each([
    ['filed elsewhere', { path: 'stencils/Beast.md' }],
    ['titled otherwise', { title: 'Beast' }],
    ['with a field more', { fields: ['Name', 'Height', 'Weight'] }],
    ['with a field renamed', { fields: ['Name', 'Weight'] }],
  ])('is not so where the second stencil is %s', (_what, override) => {
    expect(areOffersEqual([offer(), offer()], [offer(), offer(override)])).toBe(false)
  })

  it('is not so where one listing has a stencil more', () => {
    expect(areOffersEqual([offer()], [offer(), offer()])).toBe(false)
  })
})
