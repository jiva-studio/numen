/**
 * One card drawn on its own, which is what a caller laying out its own tiles
 * draws: what it says of its place among the tiles, and the words it asks for.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Card from './Card.vue'
import { grid, type DeckSection, type CardWords, type DeckCard, type Tile } from './deck'
import type { Stencil } from './stencil'

const CUTS: readonly Stencil[] = [{ name: 'Animal', fields: ['Name', 'Height'] }]

const CARDS: readonly DeckCard[] = [
  { id: 'llama', section: null, stencil: 'Animal', filled: [] },
  { id: 'yak', section: null, stencil: 'Animal', filled: [] },
]

/** The words one card is drawn with, which say nothing of adding cards. */
const WORDS: CardWords = {
  remove: 'Remove',
  carry: 'Reorder',
  cut: 'Stencil',
  cardStem: 'Card',
  nothing: 'Nothing in it',
  unknown: (stencil) => (stencil === null ? 'Cut by no stencil' : `No stencil called ${stencil}`),
  wrong: 'What is wrong',
}

const tileOf = (
  id: string,
  cards: readonly DeckCard[] = CARDS,
  sections: readonly DeckSection[] = [],
): Tile => {
  const laid = grid(cards, sections, CUTS, null)
    .runs.flatMap((run) => run.tiles)
    .find((tile) => tile.id === id)
  if (!laid) throw new Error(`no tile for ${id}`)
  return laid
}

const mountCard = (tile: Tile, props: Record<string, unknown> = {}) =>
  mount(Card, { attachTo: document.body, props: { tile, ...props } })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Card', () => {
  it('says its place among the tiles and how many stand with it, from the tile itself', () => {
    const held = mountCard(tileOf('yak'))
    expect(held.get('[data-card]').attributes('aria-posinset')).toBe('2')
    expect(held.get('[data-card]').attributes('aria-setsize')).toBe('3')
  })

  it('is drawn with words that say nothing of adding a card', () => {
    const held = mountCard(tileOf('llama'), { words: WORDS })
    expect(held.get('[data-cut-of]').attributes('aria-label')).toBe('Stencil')
  })

  it('announces a card by the place it stands in the deck, and by no name of its own', () => {
    const held = mountCard(tileOf('yak'), { words: WORDS })
    expect(held.get('[data-card]').attributes('aria-label')).toBe('Card 2')
    expect(held.get('[data-grip]').attributes('aria-label')).toBe('Reorder: Card 2')
  })

  it('says which section it stands under, and nothing where it stands under none', () => {
    const under: readonly DeckCard[] = [
      { id: 'llama', section: 'roots', stencil: 'Animal', filled: [] },
    ]
    const tile = tileOf('llama', under, [{ id: 'roots', name: 'Roots' }])
    expect(mountCard(tile).get('[data-card]').attributes('data-section')).toBe('roots')
    expect(mountCard(tileOf('llama')).get('[data-card]').attributes('data-section')).toBeUndefined()
  })

  it('holds what was typed, breaks and all, in the box of the first field', async () => {
    const held = mountCard(tileOf('llama'))
    const box = held.get<HTMLTextAreaElement>('[data-value="Name"]')
    const press = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })
    box.element.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(false)

    await box.setValue('Llama\nand alpaca')
    expect(held.emitted('write')).toEqual([['Name', 1, 'Llama\nand alpaca']])
  })

  it('says on the strip which stencil cut the card, and where none did, that none did', () => {
    const bare: readonly DeckCard[] = [{ id: 'bare', section: null, stencil: null, filled: [] }]
    expect(mountCard(tileOf('llama')).get('[data-cut-of]').text()).toBe('Animal')
    expect(mountCard(tileOf('bare', bare), { words: WORDS }).get('[data-cut-of]').text()).toBe(
      'Cut by no stencil',
    )
  })

  it('holds nothing against a card that names no stencil, having said so on its strip', () => {
    const bare: readonly DeckCard[] = [{ id: 'bare', section: null, stencil: null, filled: [] }]
    const held = mountCard(tileOf('bare', bare), { words: WORDS })
    expect(held.findAll('.card__objects')).toHaveLength(0)
  })

  it('says which stencil a card naming one is waiting for', () => {
    const gone: readonly DeckCard[] = [{ id: 'gone', section: null, stencil: 'Gone', filled: [] }]
    const held = mountCard(tileOf('gone', gone), { words: WORDS })
    expect(held.get('.card__objects').text()).toContain('No stencil called Gone')
  })

  it('reads a value of a card no stencil cuts, and types into none of them', () => {
    const bare: readonly DeckCard[] = [
      {
        id: 'bare',
        section: null,
        stencil: null,
        filled: [{ field: 'Question', text: 'what did I mean' }],
      },
    ]
    const held = mountCard(tileOf('bare', bare), { words: WORDS })
    expect(held.findAll('textarea')).toHaveLength(0)
    expect(held.find('.card__silence').exists()).toBe(false)
    const wrote = held.get('[data-wrote="Question"]')
    expect(wrote.text()).toBe('what did I mean')
    expect(wrote.attributes('aria-label')).toBe('Question')
  })

  it('says what is wrong with a value once, under the last box standing for it', () => {
    const twice: readonly DeckCard[] = [
      {
        id: 'twice',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Name', text: 'Vicuña' },
        ],
      },
    ]
    const held = mountCard(tileOf('twice', twice), {
      wrongUnder: new Map([['Name', ['this card writes Name twice']]]),
    })
    const said = held.findAll('[data-wrong-value="Name"]')
    expect(said).toHaveLength(1)
    expect(said[0]?.text()).toContain('this card writes Name twice')
  })

  it('holds nothing against any value of a card it was handed no marks for', () => {
    const held = mountCard(tileOf('llama'))
    expect(held.find('[data-wrong-value]').exists()).toBe(false)
  })
})
