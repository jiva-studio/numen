import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  ListModelsResponseSchema,
  ListReasoningLevelsResponseSchema,
  ModelErrorCode,
  ModelServiceCompleteResponseSchema,
  ModelServiceEmbedResponseSchema,
  ModelServiceGetEmbeddingProfileResponseSchema,
  Priority,
} from '@numen/protocol'
import { createModelClient, type ModelRpc } from './models'

const createRpc = (overrides: Partial<ModelRpc> = {}): ModelRpc => ({
  listModels: () =>
    Promise.resolve(
      create(ListModelsResponseSchema, {
        models: [
          { id: 'google/gemini-2.5-flash', title: 'Gemini 2.5 Flash', operations: ['complete'] },
          { id: 'qwen2.5:14b', title: '', operations: [] },
        ],
      }),
    ),
  listReasoningLevels: () =>
    Promise.resolve(
      create(ListReasoningLevelsResponseSchema, {
        levels: [
          { value: 'low', title: 'Low' },
          { value: 'high', title: 'High' },
        ],
      }),
    ),
  complete: () => Promise.resolve(create(ModelServiceCompleteResponseSchema, {})),
  embed: () => Promise.resolve(create(ModelServiceEmbedResponseSchema, {})),
  getEmbeddingProfile: () =>
    Promise.resolve(create(ModelServiceGetEmbeddingProfileResponseSchema, {})),
  ...overrides,
})

describe('createModelClient', () => {
  it('lists the models of a connection for an operation', async () => {
    const requests: unknown[] = []
    const client = createModelClient(
      createRpc({
        listModels: (request) => {
          requests.push(request)
          return Promise.resolve(create(ListModelsResponseSchema, { models: [{ id: 'm' }] }))
        },
      }),
    )

    const answer = await client.listModels('c00000001', 'complete')

    expect(requests).toStrictEqual([
      expect.objectContaining({ connectionId: 'c00000001', operation: 'complete' }),
    ])
    expect(answer).toStrictEqual({ ok: true, value: [{ id: 'm', title: 'm', operations: [] }] })
  })

  it('names a model by its id where the service gives no title', async () => {
    const answer = await createModelClient(createRpc()).listModels('c00000001', 'complete')

    expect(answer.ok && answer.value).toStrictEqual([
      { id: 'google/gemini-2.5-flash', title: 'Gemini 2.5 Flash', operations: ['complete'] },
      { id: 'qwen2.5:14b', title: 'qwen2.5:14b', operations: [] },
    ])
  })

  it('lists the reasoning levels of a model of a connection', async () => {
    const requests: unknown[] = []
    const client = createModelClient(
      createRpc({
        listReasoningLevels: (request) => {
          requests.push(request)
          return Promise.resolve(
            create(ListReasoningLevelsResponseSchema, { levels: [{ value: 'low', title: 'Low' }] }),
          )
        },
      }),
    )

    const answer = await client.listReasoningLevels('c00000001', 'm')

    expect(requests).toStrictEqual([
      expect.objectContaining({ connectionId: 'c00000001', model: 'm' }),
    ])
    expect(answer).toStrictEqual({ ok: true, value: [{ value: 'low', title: 'Low' }] })
  })

  it('reads each served error code as a failure with its detail', async () => {
    const codes: Array<[ModelErrorCode, string]> = [
      [ModelErrorCode.NOT_FOUND, 'not_found'],
      [ModelErrorCode.PROVIDER_UNAVAILABLE, 'provider_unavailable'],
    ]
    for (const [code, expected] of codes) {
      const client = createModelClient(
        createRpc({
          listModels: () =>
            Promise.resolve(
              create(ListModelsResponseSchema, { error: code, detail: 'the key was rejected' }),
            ),
          listReasoningLevels: () =>
            Promise.resolve(
              create(ListReasoningLevelsResponseSchema, {
                error: code,
                detail: 'the key was rejected',
              }),
            ),
        }),
      )

      expect(await client.listModels('c00000001', 'complete')).toStrictEqual({
        ok: false,
        error: { errorCode: expected, detail: 'the key was rejected' },
      })
      expect(await client.listReasoningLevels('c00000001', 'm')).toStrictEqual({
        ok: false,
        error: { errorCode: expected, detail: 'the key was rejected' },
      })
    }
  })
})

describe('createModelClient calls on models', () => {
  it('completes messages for the plugin and sends the priority and the signal', async () => {
    const requests: unknown[] = []
    const options: unknown[] = []
    const client = createModelClient(
      createRpc({
        complete: (request, callOptions) => {
          requests.push(request)
          options.push(callOptions)
          return Promise.resolve(create(ModelServiceCompleteResponseSchema, { text: 'fixed' }))
        },
      }),
    )
    const controller = new AbortController()

    const answer = await client.complete(
      'md.numen.tools.proofreading',
      { messages: [{ role: 'user', text: 'u' }], temperature: 0.4, maxTokens: 33 },
      { priority: 'interactive', signal: controller.signal },
    )

    expect(answer).toStrictEqual({ ok: true, value: { text: 'fixed' } })
    expect(requests).toStrictEqual([
      expect.objectContaining({
        pluginId: 'md.numen.tools.proofreading',
        messages: [expect.objectContaining({ role: 'user', text: 'u' })],
        temperature: 0.4,
        maxTokens: 33,
        priority: Priority.INTERACTIVE,
      }),
    ])
    expect(options).toStrictEqual([expect.objectContaining({ signal: controller.signal })])
  })

  it('sends background where the call names no priority', async () => {
    const requests: Array<{ priority?: Priority }> = []
    const client = createModelClient(
      createRpc({
        embed: (request) => {
          requests.push(request)
          return Promise.resolve(create(ModelServiceEmbedResponseSchema, {}))
        },
      }),
    )

    await client.embed('md.numen.engines.embeddings', ['a'])

    expect(requests.map((one) => one.priority)).toStrictEqual([Priority.BACKGROUND])
  })

  it('embeds texts into vectors in order', async () => {
    const client = createModelClient(
      createRpc({
        embed: () =>
          Promise.resolve(
            create(ModelServiceEmbedResponseSchema, {
              vectors: [{ values: [1, 2] }, { values: [3, 4] }],
            }),
          ),
      }),
    )

    const answer = await client.embed('md.numen.engines.embeddings', ['a', 'b'])

    expect(answer).toStrictEqual({
      ok: true,
      value: {
        vectors: [
          [1, 2],
          [3, 4],
        ],
      },
    })
  })

  it('gets the embedding profile of the plugin', async () => {
    const requests: unknown[] = []
    const client = createModelClient(
      createRpc({
        getEmbeddingProfile: (request) => {
          requests.push(request)
          return Promise.resolve(
            create(ModelServiceGetEmbeddingProfileResponseSchema, {
              model: 'te3',
              source: 'openrouter',
              pooling: 'mean',
              queryPrefix: 'q',
              documentPrefix: 'd',
              dimensions: 3,
              maxTokens: 8000,
            }),
          )
        },
      }),
    )

    const answer = await client.getEmbeddingProfile('md.numen.engines.embeddings')

    expect(requests).toStrictEqual([
      expect.objectContaining({ pluginId: 'md.numen.engines.embeddings' }),
    ])
    expect(answer).toStrictEqual({
      ok: true,
      value: {
        model: 'te3',
        source: 'openrouter',
        pooling: 'mean',
        queryPrefix: 'q',
        documentPrefix: 'd',
        dimensions: 3,
        maxTokens: 8000,
      },
    })
  })

  it('reads each served error code of a call with its detail and failures', async () => {
    const failures = [{ connectionName: 'OpenRouter', reason: 'payment required' }]
    const codes: Array<[ModelErrorCode, string]> = [
      [ModelErrorCode.NO_CONNECTION, 'no_connection'],
      [ModelErrorCode.ALL_FAILED, 'all_failed'],
      [ModelErrorCode.REJECTED, 'rejected'],
    ]
    for (const [code, expected] of codes) {
      const client = createModelClient(
        createRpc({
          complete: () =>
            Promise.resolve(
              create(ModelServiceCompleteResponseSchema, { error: code, detail: 'd', failures }),
            ),
          embed: () =>
            Promise.resolve(
              create(ModelServiceEmbedResponseSchema, { error: code, detail: 'd', failures }),
            ),
          getEmbeddingProfile: () =>
            Promise.resolve(
              create(ModelServiceGetEmbeddingProfileResponseSchema, {
                error: code,
                detail: 'd',
                failures,
              }),
            ),
        }),
      )
      const failure = { ok: false, error: { errorCode: expected, detail: 'd', failures } }

      expect(await client.complete('p', { messages: [] })).toStrictEqual(failure)
      expect(await client.embed('p', ['a'])).toStrictEqual(failure)
      expect(await client.getEmbeddingProfile('p')).toStrictEqual(failure)
    }
  })

  it('lets a call that cannot reach the core throw', async () => {
    const client = createModelClient(
      createRpc({ complete: () => Promise.reject(new Error('aborted')) }),
    )

    await expect(client.complete('p', { messages: [] })).rejects.toThrow('aborted')
  })
})

describe('createModelClient lists', () => {
  it('throws on a list answered with an error code only calls get', async () => {
    const client = createModelClient(
      createRpc({
        listModels: () =>
          Promise.resolve(create(ListModelsResponseSchema, { error: ModelErrorCode.ALL_FAILED })),
      }),
    )

    await expect(client.listModels('c00000001', 'complete')).rejects.toThrow('ALL_FAILED')
  })
})
