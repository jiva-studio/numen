import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { AskAgentResponseSchema, ToolCallSchema } from '@numen/protocol'
import type { AskAgentResponse } from '@numen/protocol'
import type { AgentStep } from '@numen/ui'

import { createAgentPort } from './agent'
import type { AgentClient } from './agent'

/** A tool call as the service sends one, working on a place or on none. */
const call = (place?: { path: string; span: { from: number; to: number } }): AskAgentResponse =>
  create(AskAgentResponseSchema, {
    step: {
      case: 'toolCall',
      value: create(ToolCallSchema, {
        tool: 'read',
        about: 'notes/Leaf mould.md',
        length: 12,
        ...place,
      }),
    },
  })

/** The service, saying what a test told it to and keeping what it was asked. */
const service = (says: AskAgentResponse[]) => {
  const requests: { request: unknown; signal: AbortSignal }[] = []
  const over: string[] = []
  const agent: AgentClient = {
    async *askAgent(request, options) {
      requests.push({ request, signal: options.signal })
      for (const step of says) yield step
    },
    async finishConversation(request) {
      over.push(request.conversation)
      return {}
    },
  }
  return { agent, requests, over }
}

/** Everything the port yields for what the service said. */
const steps = async (says: AskAgentResponse[]): Promise<AgentStep[]> => {
  const { agent } = service(says)
  const read: AgentStep[] = []
  const port = createAgentPort(agent)
  const abort = new AbortController()
  for await (const step of port.ask({
    question: 'why?',
    focus: 'notes/Leaf mould.md',
    conversation: 'one',
    signal: abort.signal,
  }))
    read.push(step)
  return read
}

describe('createAgentPort', () => {
  it('passes the question, the focus and the conversation to the service', async () => {
    const { agent, requests } = service([])
    const abort = new AbortController()
    for await (const _ of createAgentPort(agent).ask({
      question: 'why?',
      focus: 'notes/Leaf mould.md',
      conversation: 'one',
      signal: abort.signal,
    })) {
      // The port is read to the end; what it yields is checked below.
    }
    expect(requests).toEqual([
      {
        request: { prompt: 'why?', focus: 'notes/Leaf mould.md', conversation: 'one' },
        signal: abort.signal,
      },
    ])
  })

  it('reads the transcript as text', async () => {
    const response = create(AskAgentResponseSchema, { step: { case: 'text', value: 'Leaves.' } })
    expect(await steps([response])).toEqual([{ kind: 'said', text: 'Leaves.' }])
  })

  it('carries the place a call names', async () => {
    const place = { path: 'notes/Leaf mould.md', span: { from: 40, to: 48 } }
    expect(await steps([call(place)])).toEqual([
      {
        kind: 'toolCall',
        tool: 'read',
        subject: 'notes/Leaf mould.md',
        written: 12,
        place: { path: 'notes/Leaf mould.md', span: { from: 40, to: 48 } },
      },
    ])
  })

  it('leaves out the place of a call naming none', async () => {
    const [step] = await steps([call()])
    expect(step).toEqual({
      kind: 'toolCall',
      tool: 'read',
      subject: 'notes/Leaf mould.md',
      written: 12,
    })
    expect(step && 'place' in step).toBe(false)
  })

  it('reads the moments that carry nothing', async () => {
    const moments = [
      create(AskAgentResponseSchema, { step: { case: 'modelStarted', value: {} } }),
      create(AskAgentResponseSchema, { step: { case: 'toolFinished', value: {} } }),
    ]
    expect(await steps(moments)).toEqual([{ kind: 'thinking' }, { kind: 'answered' }])
  })

  it('reads why the agent stopped', async () => {
    const stopped = create(AskAgentResponseSchema, { step: { case: 'stopped', value: 'no key' } })
    expect(await steps([stopped])).toEqual([{ kind: 'stopped', error: 'no key' }])
  })

  it('reads a finished answer as stopped with no error', async () => {
    const stopped = create(AskAgentResponseSchema, { step: { case: 'stopped', value: 'done' } })
    expect(await steps([stopped])).toEqual([{ kind: 'stopped', error: '' }])
  })

  it('says nothing for a step it does not know', async () => {
    expect(await steps([create(AskAgentResponseSchema, {})])).toEqual([])
  })

  it('finishes the conversation it was given', async () => {
    const { agent, over } = service([])
    await createAgentPort(agent).finish('one')
    expect(over).toEqual(['one'])
  })
})

describe('the depth of reasoning', () => {
  const ask = async (reasoning?: string) => {
    const { agent, requests } = service([])
    const port = createAgentPort(agent)
    for await (const step of port.ask({
      question: 'why?',
      focus: '',
      conversation: 'one',
      signal: new AbortController().signal,
      agentId: 'alpha',
      modelId: 'deep',
      ...(reasoning === undefined ? {} : { reasoning }),
    }))
      void step
    return requests[0]?.request
  }

  it('is handed to the service where one is named', async () => {
    expect(await ask('high')).toMatchObject({ model: 'deep', reasoning: 'high' })
  })

  it('is left out where none is named', async () => {
    expect(await ask('')).not.toHaveProperty('reasoning')
    expect(await ask()).not.toHaveProperty('reasoning')
  })
})
