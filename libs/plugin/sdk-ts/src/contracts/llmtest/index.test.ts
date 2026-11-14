import { describe, expect, it } from 'vitest'
import {
  chat,
  complete,
  embed,
  getProfile,
  rerank,
  type ChatFrame,
  type ChatRequest,
  type RerankRequest,
} from '../llm'
import { createLlmHost } from './index'

const collect = async (frames: AsyncGenerator<ChatFrame, unknown>) => {
  const texts: string[] = []
  for (;;) {
    const next = await frames.next()
    if (next.done === true) return { texts, end: next.value }
    if (next.value.event.case === 'textDelta') texts.push(next.value.event.value)
  }
}

describe('createLlmHost', () => {
  it('passes the request to the served completion and returns its result to complete', async () => {
    const host = createLlmHost({
      complete: async (request) => ({
        ok: true,
        value: { text: `${request.messages.length} lines` },
      }),
    })

    const answer = await complete(host, { messages: [{ role: 'user', text: 'tide tables' }] })

    expect(answer).toMatchObject({ ok: true, value: { text: '1 lines' } })
  })

  it('passes the frames and the finish of the served turn to chat', async () => {
    const asked: ChatRequest[] = []
    const host = createLlmHost({
      chat: async function* (request) {
        asked.push(request)
        yield { event: { case: 'textDelta', value: 'low' } }
        yield { event: { case: 'textDelta', value: 'tide' } }
        return { ok: true, value: {} }
      },
    })

    const { texts, end } = await collect(
      chat(host, { messages: [{ role: 'user', text: 'when?' }] }),
    )

    expect(asked.map((one) => one.messages[0]?.text)).toEqual(['when?'])
    expect(texts).toEqual(['low', 'tide'])
    expect(end).toMatchObject({ ok: true })
  })

  it('returns the vectors and the profile the test serves', async () => {
    const host = createLlmHost({
      embed: async (request) => ({
        ok: true,
        value: { vectors: request.texts.map(() => ({ values: [0.5] })) },
      }),
      getProfile: async () => ({ ok: true, value: { model: 'harbor-embed', width: 1 } }),
    })

    const vectors = await embed(host, { texts: ['buoy', 'pier'] })
    const profile = await getProfile(host)

    expect(vectors.ok && vectors.value.vectors.map((one) => one.values)).toEqual([[0.5], [0.5]])
    expect(profile).toMatchObject({ ok: true, value: { model: 'harbor-embed', width: 1 } })
  })

  it('passes the request to the served rerank and returns its scores to rerank', async () => {
    const asked: RerankRequest[] = []
    const host = createLlmHost({
      rerank: async (request) => {
        asked.push(request)
        return { ok: true, value: { scores: request.documents.map((one) => one.length) } }
      },
    })

    const answer = await rerank(host, {
      query: 'tide',
      instruction: 'Find the notes about tides',
      documents: ['ebb', 'flood'],
    })

    expect(answer.ok && answer.value.scores).toEqual([3, 5])
    expect(asked.map((one) => [one.query, one.instruction, one.documents])).toEqual([
      ['tide', 'Find the notes about tides', ['ebb', 'flood']],
    ])
  })

  it('ends rerank with no_connection when the roles leave it out', async () => {
    const host = createLlmHost({})

    expect(await rerank(host, { query: 'tide', documents: ['ebb'] })).toMatchObject({
      ok: false,
      error: { error: 'no_connection' },
    })
  })

  it('returns the error the served operation ends with', async () => {
    const error = {
      error: 'stopped' as const,
      message: 'Harbor: key rejected',
      failures: [{ connectionName: 'Harbor', reason: 'key rejected', code: 'rejected_key' as const }],
    }
    const host = createLlmHost({ complete: async () => ({ ok: false, error }) })

    expect(await complete(host, { messages: [] })).toEqual({ ok: false, error })
  })

  it('ends an operation the roles leave out with no_connection', async () => {
    const host = createLlmHost({})

    expect(await complete(host, { messages: [] })).toMatchObject({
      ok: false,
      error: { error: 'no_connection' },
    })
  })
})
