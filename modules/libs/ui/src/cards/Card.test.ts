/**
 * One card drawn on its own, which is what a caller laying out its own tiles
 * draws: what it says of its place among the tiles, and the words it asks for.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Card from './Card.vue'
import { grid, type CardWords, type Cut, type Drawn, type Tile } from './model'

const CUTS: readonly Cut[] = [{ name: 'Animal', fields: ['Name', 'Height'] }]

const CARDS: readonly Drawn[] = [
  { id: 'llama', name: 'Llama', stencil: 'Animal', filled: [] },
  { id: 'yak', name: 'Yak', stencil: 'Animal', filled: [] },
]

/** The words one card is drawn with, which say nothing of adding cards. */
const WORDS: CardWords = {
  remove: 'Remove',
  carry: 'Reorder',
  cut: 'Stencil',
  name: 'Name',
  nothing: 'Nothing in it',
  unknown: (stencil) => (stencil === null ? 'Cut by no stencil' : `No stencil called ${stencil}`),
  twice: 'The card is named by this field',
  wrong: 'What is wrong',
}

const tileOf = (id: string, cards: readonly Drawn[] = CARDS): Tile => {
  const laid = grid(cards, CUTS, null).tiles.find((tile) => tile.id === id)
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

  it('says what is wrong with a value once, under the last box standing for it', () => {
    const twice: readonly Drawn[] = [
      {
        id: 'twice',
        name: 'Llama',
        stencil: 'Animal',
        filled: [{ field: 'Name', text: 'Alpaca' }],
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
