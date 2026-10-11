/**
 * A conversation about one card names the card beside every question it sends,
 * and is ended under the name it was held by.
 */
import { describe, expect, it, vi } from 'vitest'

import { createAgentPort } from './agent'
import type { CardFace } from '@/entities/card'

const { askAgent, finishConversation } = vi.hoisted(() => ({
  askAgent: vi.fn(async function* (_request: unknown, _options: unknown) {
    yield { step: { case: 'said', value: 'Because of the leaves.' } }
  }),
  finishConversation: vi.fn(async (_request: unknown) => ({})),
}))

vi.mock('@/shared/clients', () => ({
  agentService: { askAgent, finishConversation },
}))

const card: CardFace = {
  deck: 'decks/Words.md',
  section: '',
  mark: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
}

describe('the agent a card is asked about', () => {
  it('sends the mark and the face of the card beside the question, and reads the answer', async () => {
    const signal = new AbortController().signal
    const steps = []
    for await (const step of createAgentPort(card).ask(
      'why is it called that',
      'decks/Words.md',
      'card-0',
      signal,
    )) {
      steps.push(step)
    }

    expect(askAgent).toHaveBeenCalledWith(
      {
        asked: 'why is it called that',
        focus: 'decks/Words.md',
        conversation: 'card-0',
        mark: '3f4g5h6j7k',
        face: 'Say it',
      },
      { signal },
    )
    expect(steps).toEqual([{ kind: 'said', text: 'Because of the leaves.' }])
  })

  it('ends the conversation under the name it was held by', async () => {
    await createAgentPort(card).finish('card-0')
    expect(finishConversation).toHaveBeenCalledWith({ conversation: 'card-0' })
  })
})
