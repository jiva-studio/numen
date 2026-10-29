/**
 * One card rendered on its own, as a caller laying out its own tiles renders
 * it: its position among the tiles, and the text it requests.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import Card from './Card.vue'
import { DECK_CONTEXT } from '../../../model/context'
import { type DeckSection, type CardWords, type DeckCard } from '../../../lib/deck'
import { getGrid, type Tile } from '../../../lib/grid'
import type { CardModel } from '../../../lib/card'

const CUTS: readonly CardModel[] = [{ name: 'Animal', fields: ['Name', 'Height'] }]

const CARDS: readonly DeckCard[] = [
  { id: 'llama', section: null, cardModel: 'Animal', filled: [] },
  { id: 'yak', section: null, cardModel: 'Animal', filled: [] },
]

/** The words one card is drawn with, which say nothing of adding cards. */
const WORDS: CardWords = {
  remove: 'Remove',
  drag: 'Reorder',
  cut: 'Card model',
  cardStem: 'Card',
  nothing: 'Nothing in it',
  unknown: (cardModel) =>
    cardModel === null ? 'Cut by no card model' : `No card model called ${cardModel}`,
  wrong: 'What is wrong',
}

const createTile = (
  id: string,
  cards: readonly DeckCard[] = CARDS,
  sections: readonly DeckSection[] = [],
): Tile => {
  const cardGrid = getGrid({ cards, sections, cardModels: CUTS }, null)
    .runs.flatMap((run) => run.tiles)
    .find((tile) => tile.id === id)
  if (!cardGrid) throw new Error(`no tile for ${id}`)
  return cardGrid
}

const write =
  vi.fn<(card: string, target: { field: string; index: number }, text: string) => void>()

const mountCard = (tile: Tile, props: Record<string, unknown> = {}) =>
  mount(Card, {
    attachTo: document.body,
    props: { tile, ...props },
    global: { provide: { [DECK_CONTEXT as symbol]: { write } } },
  })

afterEach(() => {
  document.body.innerHTML = ''
  write.mockReset()
})

describe('Card', () => {
  it('says its place among the tiles and how many stand with it, from the tile itself', () => {
    const card = mountCard(createTile('yak'))
    expect(card.get('[data-card]').attributes('aria-posinset')).toBe('2')
    expect(card.get('[data-card]').attributes('aria-setsize')).toBe('3')
  })

  it('is drawn with words that say nothing of adding a card', () => {
    const card = mountCard(createTile('llama'), { words: WORDS })
    expect(card.get('[data-cut-of]').attributes('aria-label')).toBe('Card model')
  })

  it('announces a card by the place it remains in the deck, and by no name of its own', () => {
    const card = mountCard(createTile('yak'), { words: WORDS })
    expect(card.get('[data-card]').attributes('aria-label')).toBe('Card 2')
    expect(card.get('.card-header__grip').attributes('aria-label')).toBe('Reorder: Card 2')
  })

  it('says which section it is placed under, and nothing where it is placed under none', () => {
    const underCards: readonly DeckCard[] = [
      { id: 'llama', section: 'roots', cardModel: 'Animal', filled: [] },
    ]
    const tile = createTile('llama', underCards, [{ id: 'roots', name: 'Roots' }])
    expect(mountCard(tile).get('[data-card]').attributes('data-section')).toBe('roots')
    expect(
      mountCard(createTile('llama')).get('[data-card]').attributes('data-section'),
    ).toBeUndefined()
  })

  it('holds what was typed, breaks and all, in the box of the first field', async () => {
    const card = mountCard(createTile('llama'))
    const box = card.get<HTMLTextAreaElement>('[data-value="Name"]')
    const press = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })
    box.element.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(false)

    await box.setValue('Llama\nand alpaca')
    expect(write.mock.calls).toEqual([['llama', { field: 'Name', index: 1 }, 'Llama\nand alpaca']])
  })

  it('says on the strip which card model cut the card, and where none did, that none did', () => {
    const bareCards: readonly DeckCard[] = [
      { id: 'bare', section: null, cardModel: null, filled: [] },
    ]
    expect(mountCard(createTile('llama')).get('[data-cut-of]').text()).toBe('Animal')
    expect(
      mountCard(createTile('bare', bareCards), { words: WORDS }).get('[data-cut-of]').text(),
    ).toBe('Cut by no card model')
  })

  it('holds nothing against a card that names no card model, having returned so on its strip', () => {
    const bareCards: readonly DeckCard[] = [
      { id: 'bare', section: null, cardModel: null, filled: [] },
    ]
    const card = mountCard(createTile('bare', bareCards), { words: WORDS })
    expect(card.findAll('.card__objections')).toHaveLength(0)
  })

  it('says which card model a card naming one is waiting for', () => {
    const removed: readonly DeckCard[] = [
      { id: 'gone', section: null, cardModel: 'Gone', filled: [] },
    ]
    const card = mountCard(createTile('gone', removed), { words: WORDS })
    expect(card.get('.card__objections').text()).toContain('No card model called Gone')
  })

  it('reads a value of a card no card model cuts, and types into none of them', () => {
    const bareCards: readonly DeckCard[] = [
      {
        id: 'bare',
        section: null,
        cardModel: null,
        filled: [{ field: 'Question', text: 'what did I mean' }],
      },
    ]
    const card = mountCard(createTile('bare', bareCards), { words: WORDS })
    expect(card.findAll('textarea')).toHaveLength(0)
    expect(card.find('.card__silence').exists()).toBe(false)
    const wrote = card.get('[data-wrote="Question"]')
    expect(wrote.text()).toBe('what did I mean')
    expect(wrote.attributes('aria-label')).toBe('Question')
  })

  it('says what is wrong with a value once, under the last box standing for it', () => {
    const twice: readonly DeckCard[] = [
      {
        id: 'twice',
        section: null,
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const card = mountCard(createTile('twice', twice), {
      wrongUnder: new Map([['Name', ['this card writes Name twice']]]),
    })
    const response = card.findAll('[data-wrong-value="Name"]')
    expect(response).toHaveLength(1)
    expect(response[0]?.text()).toContain('this card writes Name twice')
  })

  it('holds nothing against any value of a card it was handed no marks for', () => {
    const card = mountCard(createTile('llama'))
    expect(card.find('[data-wrong-value]').exists()).toBe(false)
  })
})
