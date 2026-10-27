/**
 * A conversation about one card names the card in the context of every
 * question it sends, and is ended under the name it was held by.
 */
import { describe, expect, it } from 'vitest'
import { createMockAgentHost } from '@numen/plugin-sdk/testing'

import { createCardAgentPort } from './agent'
import type { ReviewCard } from '../../../entities/card'

const card: ReviewCard = {
  deck: 'decks/Words.md',
  section: '',
  cardId: '3f4g5h6j7k',
  template: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
}

describe('createCardAgentPort', () => {
  it('sends the vault, and the deck, the card and the review card in the query context', async () => {
    const agent = createMockAgentHost([{ kind: 'said', text: 'Because of the leaves.' }])
    const signal = new AbortController().signal
    const steps = []
    for await (const step of createCardAgentPort(agent, card, 'v-garden').ask({
      question: 'why is it called that',
      focus: 'decks/Words.md',
      conversation: 'card-0',
      signal,
    })) {
      steps.push(step)
    }

    expect(agent.getQueries()).toEqual([
      {
        question: 'why is it called that',
        focus: 'decks/Words.md',
        conversation: 'card-0',
        signal,
        vault: 'v-garden',
        context: {
          'md.numen.views.flashcards.studio/deck': 'decks/Words.md',
          'md.numen.views.flashcards.studio/card': '3f4g5h6j7k',
          'md.numen.views.flashcards.studio/template': 'Say it',
        },
      },
    ])
    expect(steps).toEqual([{ kind: 'said', text: 'Because of the leaves.' }])
  })

  it('finishes the conversation by its id', async () => {
    const agent = createMockAgentHost()
    await createCardAgentPort(agent, card, 'v-garden').finish('card-0')
    expect(agent.getFinished()).toEqual(['card-0'])
  })
})
