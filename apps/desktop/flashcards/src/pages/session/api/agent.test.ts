/**
 * A conversation about one card names the card beside every question it sends,
 * and is ended under the name it was held by.
 */
import { describe, expect, it, vi } from 'vitest'

import { createCardAgentPort } from './agent'
import type { CardFace } from '@/entities/card'

const { askAgent, finishConversation } = vi.hoisted(() => ({
  askAgent: vi.fn(async function* (_request: unknown, _options: unknown) {
    yield { step: { case: 'text', value: 'Because of the leaves.' } }
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

describe('the agent a card is requested about', () => {
  it('sends the mark and the face of the card alongside the question, and reads the answer', async () => {
    const signal = new AbortController().signal
    const steps = []
    for await (const step of createCardAgentPort(card).ask({
      question: 'why is it called that',
      focus: 'decks/Words.md',
      conversation: 'card-0',
      signal,
    })) {
      steps.push(step)
    }

    expect(askAgent).toHaveBeenCalledWith(
      {
        prompt: 'why is it called that',
        focus: 'decks/Words.md',
        conversation: 'card-0',
        mark: '3f4g5h6j7k',
        face: 'Say it',
      },
      { signal },
    )
    expect(steps).toEqual([{ kind: 'said', text: 'Because of the leaves.' }])
  })

  it('ends the conversation under the name it was retained by', async () => {
    await createCardAgentPort(card).finish('card-0')
    expect(finishConversation).toHaveBeenCalledWith({ conversation: 'card-0' })
  })
})
