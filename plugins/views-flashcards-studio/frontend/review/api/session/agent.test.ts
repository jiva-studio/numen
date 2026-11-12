/**
 * A conversation about one card hands the card to the tools the agent calls.
 */
import { describe, expect, it } from 'vitest'

import { getCardToolContext } from './agent'
import type { ReviewCard } from '../../lib/card/cards'

const card: ReviewCard = {
  address: { deck: 'decks/Words.md', section: '', cardId: '3f4g5h6j7k', template: 'Say it' },
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
}

describe('getCardToolContext', () => {
  it('names the deck, the card and the template under the keys of the plugin', () => {
    expect(getCardToolContext(card)).toStrictEqual({
      'md.numen.views.flashcards.studio/deck': 'decks/Words.md',
      'md.numen.views.flashcards.studio/card': '3f4g5h6j7k',
      'md.numen.views.flashcards.studio/template': 'Say it',
    })
  })
})
