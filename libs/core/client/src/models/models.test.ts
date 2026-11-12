import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  HostModelChatFinishReason,
  ListModelsResponseSchema,
  ListReasoningLevelsResponseSchema,
  ModelErrorCode,
  ModelServiceChatResponseSchema,
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
  chat: () =>
    (async function* () {
      yield* []
    })(),
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

type ChatEventCase = Parameters<typeof create<typeof ModelServiceChatResponseSchema>>[1]

const streamOf = (...answers: ChatEventCase[]) =>
  (async function* () {
    for (const answer of answers) yield create(ModelServiceChatResponseSchema, answer)
  })()

const readChat = async (stream: AsyncGenerator<unknown, unknown>) => {
  const events: unknown[] = []
  for (;;) {
    const next = await stream.next()
    if (next.done === true) return { events, end: next.value }
    events.push(next.value)
  }
}

describe('createModelClient chat', () => {
  it('streams text, tool calls and the finish of a turn in order', async () => {
    const client = createModelClient(
      createRpc({
        chat: () =>
          streamOf(
            { event: { case: 'textDelta', value: 'Let me ' } },
            { event: { case: 'textDelta', value: 'look.' } },
            {
              event: {
                case: 'toolCall',
                value: { id: 'c1', name: 'list_notes', argumentsJson: '{"limit":3}' },
              },
            },
            {
              event: {
                case: 'finish',
                value: { reason: HostModelChatFinishReason.TOOL_CALLS },
              },
            },
          ),
      }),
    )

    const { events, end } = await readChat(client.chat('md.numen.agents.numen', { messages: [] }))

    expect(events).toStrictEqual([
      { textDelta: 'Let me ' },
      { textDelta: 'look.' },
      { toolCall: { id: 'c1', name: 'list_notes', arguments: '{"limit":3}' } },
      { finish: { reason: 'tool_calls' } },
    ])
    expect(end).toBeNull()
  })

  it('sends the turn of the plugin with the history, the tools, the priority and the signal', async () => {
    const requests: unknown[] = []
    const options: unknown[] = []
    const client = createModelClient(
      createRpc({
        chat: (request, callOptions) => {
          requests.push(request)
          options.push(callOptions)
          return streamOf({
            event: { case: 'finish', value: { reason: HostModelChatFinishReason.STOP } },
          })
        },
      }),
    )
    const controller = new AbortController()

    await readChat(
      client.chat(
        'md.numen.agents.numen',
        {
          messages: [
            { role: 'user', text: 'list' },
            {
              role: 'assistant',
              text: '',
              toolCalls: [{ id: 'c0', name: 'list_notes', arguments: '{}' }],
            },
            { role: 'tool', text: 'a, b', toolCallId: 'c0' },
          ],
          tools: [{ name: 'list_notes', description: 'Lists.', parameters: { type: 'object' } }],
          temperature: 0.4,
          maxTokens: 33,
        },
        { priority: 'interactive', signal: controller.signal },
      ),
    )

    expect(requests).toStrictEqual([
      expect.objectContaining({
        pluginId: 'md.numen.agents.numen',
        messages: [
          expect.objectContaining({ role: 'user', text: 'list' }),
          expect.objectContaining({
            role: 'assistant',
            toolCalls: [
              expect.objectContaining({ id: 'c0', name: 'list_notes', argumentsJson: '{}' }),
            ],
          }),
          expect.objectContaining({ role: 'tool', text: 'a, b', toolCallId: 'c0' }),
        ],
        tools: [
          expect.objectContaining({
            name: 'list_notes',
            description: 'Lists.',
            parametersJson: '{"type":"object"}',
          }),
        ],
        temperature: 0.4,
        maxTokens: 33,
        priority: Priority.INTERACTIVE,
      }),
    ])
    expect(options).toStrictEqual([expect.objectContaining({ signal: controller.signal })])
  })

  it('ends the turn with the failure the core serves and keeps the text before it', async () => {
    const client = createModelClient(
      createRpc({
        chat: () =>
          streamOf(
            { event: { case: 'textDelta', value: 'partial' } },
            {
              event: {
                case: 'failure',
                value: {
                  error: ModelErrorCode.ALL_FAILED,
                  detail: 'Ollama: unreachable',
                  failures: [{ connectionName: 'Ollama', reason: 'unreachable' }],
                },
              },
            },
          ),
      }),
    )

    const { events, end } = await readChat(client.chat('md.numen.agents.numen', { messages: [] }))

    expect(events).toStrictEqual([{ textDelta: 'partial' }])
    expect(end).toStrictEqual({
      errorCode: 'all_failed',
      detail: 'Ollama: unreachable',
      failures: [{ connectionName: 'Ollama', reason: 'unreachable' }],
    })
  })

  it('reads a plugin with no row as no_connection and a wrong request as rejected', async () => {
    const codes: Array<[ModelErrorCode, string]> = [
      [ModelErrorCode.NO_CONNECTION, 'no_connection'],
      [ModelErrorCode.REJECTED, 'rejected'],
    ]
    for (const [code, expected] of codes) {
      const client = createModelClient(
        createRpc({
          chat: () => streamOf({ event: { case: 'failure', value: { error: code, detail: 'd' } } }),
        }),
      )

      const { end } = await readChat(client.chat('md.numen.agents.numen', { messages: [] }))

      expect(end).toStrictEqual({ errorCode: expected, detail: 'd', failures: [] })
    }
  })

  it('throws where the stream ends before the turn did', async () => {
    const client = createModelClient(
      createRpc({ chat: () => streamOf({ event: { case: 'textDelta', value: 'cut' } }) }),
    )

    await expect(readChat(client.chat('md.numen.agents.numen', { messages: [] }))).rejects.toThrow(
      'before the turn did',
    )
  })
})
