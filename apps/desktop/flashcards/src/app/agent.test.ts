import { describe, expect, it } from 'vitest'
import type { AskAgentResponse } from '@numen/protocol'
import type { LinkClient } from '@numen/wire'
import { createWindowAgent, type WindowAgentClient } from './agent'
import { WORDS } from '@/shared/words'

/** The note service, answering one link of the garden vault and keeping every request. */
const linkRequests: unknown[] = []
const links: LinkClient = {
  resolveAddresses: async (request) => {
    linkRequests.push(request)
    return {
      found: [{ raw: '[[Peat]]', address: 'name://Peat', text: 'Peat' }],
      resolved: [{ written: 'name://Peat', path: 'Peat.md', isCrossed: false }],
    }
  },
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** The agent service, answering the state given and keeping every request. */
const createClient = (state: () => Promise<{ unreachable: string }>) => {
  const requests: unknown[] = []
  const signals: AbortSignal[] = []
  const client: WindowAgentClient = {
    getAgentState: (_request, options) => {
      signals.push(options.signal)
      return state()
    },
    async *askAgent(request): AsyncIterable<AskAgentResponse> {
      requests.push(request)
      yield* [] as AskAgentResponse[]
    },
    finishConversation: async () => ({}),
  }
  return { client, requests, signals }
}

describe('createWindowAgent', () => {
  it('sends the query context to the service', async () => {
    const { client, requests } = createClient(async () => ({ unreachable: '' }))
    const { agent } = createWindowAgent(client, links)

    for await (const step of agent.ask({
      question: 'what is this?',
      focus: 'decks/Words.md',
      conversation: 'card-0',
      signal: new AbortController().signal,
      context: { 'md.example.cards/card': 'k7' },
    }))
      void step

    expect(requests).toMatchObject([{ context: { 'md.example.cards/card': 'k7' } }])
  })

  it('reports the agent reachable until the state arrives, then its reason', async () => {
    const { client } = createClient(async () => ({ unreachable: 'no model is set up' }))
    const { agent } = createWindowAgent(client, links)
    expect(agent.getUnreachable()).toBe('')

    await settle()

    expect(agent.getUnreachable()).toBe('no model is set up')
  })

  it('reports the agent unreachable when the state request fails', async () => {
    const { client } = createClient(async () => {
      throw new Error('the connection went')
    })
    const { agent } = createWindowAgent(client, links)

    await settle()

    expect(agent.getUnreachable()).toBe(WORDS.agentUnreachable)
  })

  it('aborts the state request on dispose', () => {
    const { client, signals } = createClient(() => new Promise<never>(() => {}))
    const { dispose } = createWindowAgent(client, links)

    dispose()

    expect(signals[0]?.aborted).toBe(true)
  })

  it('resolves the wikilinks of a text in the vault named', async () => {
    const { client } = createClient(async () => ({ unreachable: '' }))
    const { agent } = createWindowAgent(client, links)

    const found = await agent.resolveLinks('Unlike [[Peat]].', 'garden')

    expect(found).toEqual([
      { raw: '[[Peat]]', address: 'name://Peat', text: 'Peat', path: 'Peat.md' },
    ])
    expect(linkRequests.at(-1)).toEqual({ from: '', text: 'Unlike [[Peat]].', vault: 'garden' })
  })
})
