/**
 * One card drawn on its own, which is what a caller laying out its own tiles
 * draws: what it says of its place among the tiles, and the words it asks for.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Card from './Card.vue'
import { grid, type Banded, type CardWords, type Drawn, type Tile } from './deck'
import type { Cut } from './stencil'

const CUTS: readonly Cut[] = [{ name: 'Animal', fields: ['Name', 'Height'] }]

const CARDS: readonly Drawn[] = [
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
  cards: readonly Drawn[] = CARDS,
  sections: readonly Banded[] = [],
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
    const under: readonly Drawn[] = [
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

  it('says what is wrong with a value once, under the last box standing for it', () => {
    const twice: readonly Drawn[] = [
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
