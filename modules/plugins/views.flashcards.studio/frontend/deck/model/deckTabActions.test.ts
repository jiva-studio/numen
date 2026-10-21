/**
 * What a deck tab does to the deck it holds, handed on to be stored.
 */
import { describe, expect, it, vi } from 'vitest'
import { dropCard, type BufferCard, type BufferDeck } from '../lib/deck'
import { createDeckTabActions } from './deckTabActions'

const card = (id: string): BufferCard => ({
  id,
  mark: '',
  section: null,
  heading: id,
  stencilLink: 'Animal',
  stencilPath: 'Animal.md',
  preamble: '',
  values: [],
})

const DECK: BufferDeck = { preamble: '', cards: [card('a'), card('b')], sections: [], tail: '' }

describe('a card moved', () => {
  it('is stored as the deck with the card let go where it was dropped', () => {
    const updateDeck = vi.fn<(id: string, deck: BufferDeck) => void>()
    const actions = createDeckTabActions('tab', {
      getDeck: () => DECK,
      updateDeck,
      getOffers: () => [],
    })

    actions.moveCard('a', null)

    expect(updateDeck).toHaveBeenCalledWith('tab', dropCard(DECK, 'a', null))
    expect(updateDeck.mock.calls[0]?.[1].cards.map((one) => one.id)).toStrictEqual(['b', 'a'])
  })
})
