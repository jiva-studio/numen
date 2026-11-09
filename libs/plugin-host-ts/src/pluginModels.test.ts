import { describe, expect, it } from 'vitest'
import type { ModelClient } from '@numen/core-client'
import { createPluginModels } from './pluginModels'

const failure = { ok: false, error: { errorCode: 'no_connection', detail: '', failures: [] } }

const createClient = (overrides: Partial<ModelClient> = {}): ModelClient =>
  ({
    complete: () => Promise.resolve({ ok: true, value: { text: 'fixed' } }),
    embed: () => Promise.resolve({ ok: true, value: { vectors: [[1, 2]] } }),
    getEmbeddingProfile: () => Promise.reject(new Error('not used')),
    chat: () => {
      throw new Error('not used')
    },
    ...overrides,
  }) as unknown as ModelClient

describe('the model calls of a plugin', () => {
  it('completes messages for the plugin and returns the text', async () => {
    const requests: unknown[] = []
    const models = createPluginModels(
      createClient({
        complete: (pluginId, request, options) => {
          requests.push({ pluginId, request, options })
          return Promise.resolve({ ok: true, value: { text: 'fixed' } })
        },
      }),
      'md.numen.tools.proofreading',
    )
    const controller = new AbortController()

    const answer = await models.complete(
      { messages: [{ role: 'user', text: 'u' }], temperature: 0.4, maxTokens: 100 },
      { priority: 'interactive', signal: controller.signal },
    )

    expect(answer).toStrictEqual({ ok: true, value: { text: 'fixed' } })
    expect(requests).toStrictEqual([
      {
        pluginId: 'md.numen.tools.proofreading',
        request: { messages: [{ role: 'user', text: 'u' }], temperature: 0.4, maxTokens: 100 },
        options: { priority: 'interactive', signal: controller.signal },
      },
    ])
  })

  it('embeds texts into vectors in order', async () => {
    const requests: unknown[] = []
    const models = createPluginModels(
      createClient({
        embed: (pluginId, texts, options) => {
          requests.push({ pluginId, texts, options })
          return Promise.resolve({ ok: true, value: { vectors: [[1], [2]] } })
        },
      }),
      'md.numen.engines.embeddings',
    )

    const answer = await models.embed({ texts: ['a', 'b'] }, { priority: 'background' })

    expect(answer).toStrictEqual({ ok: true, value: { vectors: [[1], [2]] } })
    expect(requests).toStrictEqual([
      {
        pluginId: 'md.numen.engines.embeddings',
        texts: ['a', 'b'],
        options: { priority: 'background' },
      },
    ])
  })

  it('returns the embedding profile of the row that answered', async () => {
    const profile = {
      model: 'te3',
      source: 'openrouter',
      pooling: 'mean',
      queryPrefix: 'q',
      documentPrefix: 'd',
      dimensions: 3,
      maxTokens: 8000,
    }
    const models = createPluginModels(
      createClient({ getEmbeddingProfile: () => Promise.resolve({ ok: true, value: profile }) }),
      'md.numen.engines.embeddings',
    )

    expect(await models.getEmbeddingProfile()).toStrictEqual({ ok: true, value: profile })
  })

  it('reports a plugin with no row as no_connection', async () => {
    const models = createPluginModels(
      createClient({ complete: () => Promise.resolve(failure as never) }),
      'md.numen.a',
    )

    expect(await models.complete({ messages: [] })).toStrictEqual({
      ok: false,
      error: { error: 'no_connection', message: '', failures: [] },
    })
  })

  it('reports every failed row by connection name and reason', async () => {
    const rows = [
      { connectionName: 'OpenRouter', reason: 'payment required' },
      { connectionName: 'Ollama', reason: 'unreachable' },
    ]
    const models = createPluginModels(
      createClient({
        embed: () =>
          Promise.resolve({
            ok: false,
            error: {
              errorCode: 'all_failed',
              detail: 'OpenRouter: payment required; Ollama: unreachable',
              failures: rows,
            },
          }),
      }),
      'md.numen.a',
    )

    expect(await models.embed({ texts: ['a'] })).toStrictEqual({
      ok: false,
      error: {
        error: 'all_failed',
        message: 'OpenRouter: payment required; Ollama: unreachable',
        failures: rows,
      },
    })
  })

  it('reports a request a connection found wrong as rejected', async () => {
    const models = createPluginModels(
      createClient({
        complete: () =>
          Promise.resolve({
            ok: false,
            error: {
              errorCode: 'rejected',
              detail: 'OpenRouter: bad request',
              failures: [{ connectionName: 'OpenRouter', reason: 'bad request' }],
            },
          }),
      }),
      'md.numen.a',
    )

    const answer = await models.complete({ messages: [] })

    expect(!answer.ok && answer.error.error).toBe('rejected')
    expect(!answer.ok && answer.error.failures).toStrictEqual([
      { connectionName: 'OpenRouter', reason: 'bad request' },
    ])
  })

  it('reports a call whose signal aborted as canceled', async () => {
    const controller = new AbortController()
    const models = createPluginModels(
      createClient({
        complete: () => {
          controller.abort()
          return Promise.reject(new Error('aborted'))
        },
      }),
      'md.numen.a',
    )

    expect(await models.complete({ messages: [] }, { signal: controller.signal })).toStrictEqual({
      ok: false,
      error: { error: 'canceled', message: 'aborted', failures: [] },
    })
  })

  it('lets a failure that is not a cancellation throw', async () => {
    const models = createPluginModels(
      createClient({ embed: () => Promise.reject(new Error('transport down')) }),
      'md.numen.a',
    )

    await expect(models.embed({ texts: ['a'] })).rejects.toThrow('transport down')
  })
})

const drain = async (stream: AsyncGenerator<unknown, unknown>) => {
  const events: unknown[] = []
  for (;;) {
    const next = await stream.next()
    if (next.done === true) return { events, end: next.value }
    events.push(next.value)
  }
}

describe('the chat of a plugin', () => {
  it('streams the events of the turn for the plugin and returns null at the end', async () => {
    const requests: unknown[] = []
    const models = createPluginModels(
      createClient({
        chat: async function* (pluginId, request, options) {
          requests.push({ pluginId, request, options })
          yield { textDelta: 'Hi' }
          yield { finish: { reason: 'stop' } }
          return null
        },
      }),
      'md.numen.agents.numen',
    )

    const { events, end } = await drain(
      models.chat({ messages: [{ role: 'user', text: 'u' }] }, { priority: 'interactive' }),
    )

    expect(events).toStrictEqual([{ textDelta: 'Hi' }, { finish: { reason: 'stop' } }])
    expect(end).toBeNull()
    expect(requests).toStrictEqual([
      {
        pluginId: 'md.numen.agents.numen',
        request: { messages: [{ role: 'user', text: 'u' }] },
        options: { priority: 'interactive' },
      },
    ])
  })

  it('returns the failure of the turn as the error of the model, after the text before it', async () => {
    const rows = [{ connectionName: 'Ollama', reason: 'unreachable' }]
    const models = createPluginModels(
      createClient({
        chat: async function* () {
          yield { textDelta: 'partial' }
          return { errorCode: 'all_failed', detail: 'Ollama: unreachable', failures: rows }
        },
      }),
      'md.numen.agents.numen',
    )

    const { events, end } = await drain(models.chat({ messages: [] }))

    expect(events).toStrictEqual([{ textDelta: 'partial' }])
    expect(end).toStrictEqual({
      error: 'all_failed',
      message: 'Ollama: unreachable',
      failures: rows,
    })
  })

  it('returns a turn whose signal aborted as canceled', async () => {
    const controller = new AbortController()
    const models = createPluginModels(
      createClient({
        chat: async function* () {
          yield { textDelta: 'partial' }
          controller.abort()
          throw new Error('aborted')
        },
      }),
      'md.numen.agents.numen',
    )

    const { events, end } = await drain(
      models.chat({ messages: [] }, { signal: controller.signal }),
    )

    expect(events).toStrictEqual([{ textDelta: 'partial' }])
    expect(end).toStrictEqual({ error: 'canceled', message: 'aborted', failures: [] })
  })

  it('lets a failure that is not a cancellation throw', async () => {
    const models = createPluginModels(
      createClient({
        chat: async function* () {
          yield { textDelta: 'partial' }
          throw new Error('transport down')
        },
      }),
      'md.numen.agents.numen',
    )

    await expect(drain(models.chat({ messages: [] }))).rejects.toThrow('transport down')
  })
})
