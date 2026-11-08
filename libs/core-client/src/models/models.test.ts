import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  ListModelsResponseSchema,
  ListReasoningLevelsResponseSchema,
  ModelErrorCode,
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

    expect(requests).toStrictEqual([expect.objectContaining({ connectionId: 'c00000001', operation: 'complete' })])
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
          return Promise.resolve(create(ListReasoningLevelsResponseSchema, { levels: [{ value: 'low', title: 'Low' }] }))
        },
      }),
    )

    const answer = await client.listReasoningLevels('c00000001', 'm')

    expect(requests).toStrictEqual([expect.objectContaining({ connectionId: 'c00000001', model: 'm' })])
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
          listModels: () => Promise.resolve(create(ListModelsResponseSchema, { error: code, detail: 'the key was rejected' })),
          listReasoningLevels: () =>
            Promise.resolve(create(ListReasoningLevelsResponseSchema, { error: code, detail: 'the key was rejected' })),
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
