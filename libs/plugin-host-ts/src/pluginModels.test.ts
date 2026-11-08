import { describe, expect, it } from 'vitest'
import type { ModelClient } from '@numen/core-client'
import { createPluginModels } from './pluginModels'

const failure = { ok: false, error: { errorCode: 'no_connection', detail: '', failures: [] } }

const createClient = (overrides: Partial<ModelClient> = {}): ModelClient =>
  ({
    complete: () => Promise.resolve({ ok: true, value: { text: 'fixed' } }),
    embed: () => Promise.resolve({ ok: true, value: { vectors: [[1, 2]] } }),
    getEmbeddingProfile: () => Promise.reject(new Error('not used')),
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
