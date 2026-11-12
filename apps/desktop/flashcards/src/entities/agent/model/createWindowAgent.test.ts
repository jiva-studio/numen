import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreHandlers } from '@numen/core-client/testing'
import { WORDS } from '../words'
import { createWindowAgent } from './createWindowAgent'

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const core = {
  getAgentState: vi.fn(),
  askAgent: vi.fn(),
  finishConversation: vi.fn(),
  resolveAddresses: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  core.getAgentState.mockResolvedValue({ unreachable: '' })
  core.askAgent.mockImplementation(async function* () {
    yield* []
  })
  core.finishConversation.mockResolvedValue({})
  restore = installCoreHandlers(core)
})

afterEach(() => {
  restore()
})

describe('createWindowAgent', () => {
  it('sends the query context to the service', async () => {
    const { agent } = createWindowAgent()

    for await (const _ of agent.ask({
      question: 'what is this?',
      focus: 'decks/Words.md',
      conversation: 'card-0',
      signal: new AbortController().signal,
      context: { 'md.example.cards/card': 'k7' },
    })) {
      // The port is read to the end; the request is checked below.
    }

    expect(core.askAgent.mock.calls.map((call) => call[0])).toMatchObject([
      { context: { 'md.example.cards/card': 'k7' } },
    ])
  })

  it('reports the agent reachable until the state arrives, then its reason', async () => {
    core.getAgentState.mockResolvedValue({ unreachable: 'no model is set up' })
    const { agent } = createWindowAgent()
    expect(agent.getUnreachable()).toBe('')

    await vi.waitFor(() => {
      expect(agent.getUnreachable()).toBe('no model is set up')
    })
  })

  it('reports the agent unreachable when the state request fails', async () => {
    core.getAgentState.mockRejectedValue(new Error('the connection went'))
    const { agent } = createWindowAgent()

    await vi.waitFor(() => {
      expect(agent.getUnreachable()).toBe(WORDS.agentUnreachable)
    })
  })

  it('aborts the state request on dispose and ignores its rejection', async () => {
    let signal: AbortSignal | undefined
    core.getAgentState.mockImplementation(
      (_request: unknown, context: { signal: AbortSignal }) =>
        new Promise((_, reject) => {
          signal = context.signal
          signal.addEventListener('abort', () => reject(new Error('aborted')))
        }),
    )
    const { agent, dispose } = createWindowAgent()
    await vi.waitFor(() => {
      expect(signal).toBeDefined()
    })

    dispose()
    await settle()

    expect(signal?.aborted).toBe(true)
    expect(agent.getUnreachable()).toBe('')
  })

  it('finishes a conversation and exposes no local settings or models', async () => {
    const { agent } = createWindowAgent()

    await agent.finish('card-0')

    expect(core.finishConversation.mock.calls.map((call) => call[0])).toMatchObject([
      { conversation: 'card-0' },
    ])
    expect(agent.getSetting(['model'])).toBeUndefined()
    expect(agent.getModelsAt(['provider'])).toStrictEqual([])
    await expect(agent.writeSetting(['model'], 'claude')).rejects.toThrow(
      'this window keeps no agent settings',
    )
  })

  it('resolves the wikilinks of a text in the vault named', async () => {
    core.resolveAddresses.mockResolvedValue({
      found: [{ raw: '[[Peat]]', address: 'name://Peat', text: 'Peat' }],
      resolved: [{ written: 'name://Peat', path: 'Peat.md', isCrossed: false }],
    })
    const { agent } = createWindowAgent()

    const found = await agent.resolveLinks('Unlike [[Peat]].', 'garden')

    expect(found).toEqual([
      { raw: '[[Peat]]', address: 'name://Peat', text: 'Peat', path: 'Peat.md' },
    ])
    expect(core.resolveAddresses.mock.calls.at(-1)?.[0]).toMatchObject({
      from: '',
      text: 'Unlike [[Peat]].',
      vault: 'garden',
    })
  })
})
