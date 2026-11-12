import { describe, expect, it } from 'vitest'
import { create, fromJson, toJson, type JsonValue } from '@bufbuild/protobuf'
import { ValueSchema } from '@bufbuild/protobuf/wkt'
import {
  CreateProviderConnectionResponseSchema,
  DeleteProviderConnectionResponseSchema,
  ListProviderConnectionsResponseSchema,
  ListProviderConnectionTypesResponseSchema,
  ListRowOptionsResponseSchema,
  ProviderConnectionErrorCode,
  UpdateProviderConnectionResponseSchema,
} from '@numen/protocol'
import { createConnectionClient, type ConnectionRpc } from './connections'

const value = (json: JsonValue) => fromJson(ValueSchema, json)

const createRpc = (overrides: Partial<ConnectionRpc> = {}): ConnectionRpc => ({
  listProviderConnectionTypes: () =>
    Promise.resolve(
      create(ListProviderConnectionTypesResponseSchema, {
        types: [
          {
            typeId: 'md.numen.connections.llm/openrouter',
            pluginId: 'md.numen.connections.llm',
            pluginName: 'LLM connections',
            title: 'OpenRouter',
            description: 'One key for many models.',
            operations: ['llm.complete', 'llm.embed', 'llm.models'],
            isAvailable: true,
            isBuiltIn: true,
            rowFields: [
              {
                key: 'model',
                type: 'string',
                title: 'Model',
                isRequired: true,
                optionsFrom: 'llm.models',
              },
            ],
            fields: [
              {
                key: 'baseUrl',
                type: 'string',
                title: 'Address',
                isAddress: true,
                fixedValue: 'https://openrouter.ai/api/v1',
              },
              { key: 'apiKey', type: 'secret', title: 'API key', isRequired: true },
              {
                key: 'mode',
                type: 'enum',
                title: 'Mode',
                default: value('fast'),
                options: { values: [value('fast'), value({ value: 'slow', title: 'Slow' })] },
              },
              {
                key: 'parallelRequests',
                type: 'number',
                title: 'Parallel requests',
                default: value(4),
                min: 1,
                max: 32,
                step: 1,
              },
            ],
          },
        ],
        operations: [{ name: 'llm.embed', title: 'embed text' }],
      }),
    ),
  listProviderConnections: () =>
    Promise.resolve(
      create(ListProviderConnectionsResponseSchema, {
        connections: [
          {
            id: 'c00000001',
            typeId: 'md.numen.connections.llm/openrouter',
            name: 'Work',
            revision: 3,
            fields: { parallelRequests: value(8), stream: value(true), label: value('a') },
            secretKeysSet: ['apiKey'],
            isTypeAvailable: true,
            isComplete: true,
            uses: [
              {
                pluginId: 'md.numen.tools.proofreading',
                pluginName: 'Proofreading',
                operations: ['llm.complete'],
              },
            ],
          },
        ],
      }),
    ),
  createProviderConnection: () =>
    Promise.resolve(
      create(CreateProviderConnectionResponseSchema, {
        connection: { id: 'c00000002', name: 'New' },
      }),
    ),
  updateProviderConnection: () =>
    Promise.resolve(
      create(UpdateProviderConnectionResponseSchema, {
        connection: { id: 'c00000001', name: 'Renamed', revision: 4 },
      }),
    ),
  deleteProviderConnection: () =>
    Promise.resolve(create(DeleteProviderConnectionResponseSchema, {})),
  listRowOptions: () => Promise.resolve(create(ListRowOptionsResponseSchema, {})),
  ...overrides,
})

describe('createConnectionClient', () => {
  it('lists the types with the fields their forms are built from', async () => {
    const { types } = await createConnectionClient(createRpc()).listTypes()

    expect(types).toHaveLength(1)
    const [type] = types
    expect(type).toMatchObject({
      id: 'md.numen.connections.llm/openrouter',
      pluginName: 'LLM connections',
      title: 'OpenRouter',
      operations: ['llm.complete', 'llm.embed', 'llm.models'],
      isAvailable: true,
      isBuiltIn: true,
    })
    const [address, key, mode, parallel] = type?.fields ?? []
    expect(address).toMatchObject({
      key: 'baseUrl',
      isAddress: true,
      fixedValue: 'https://openrouter.ai/api/v1',
    })
    expect(key).toMatchObject({ type: 'secret', isRequired: true })
    expect(mode?.options).toStrictEqual([
      { value: 'fast', title: 'fast' },
      { value: 'slow', title: 'Slow' },
    ])
    expect(mode?.defaultValue).toBe('fast')
    expect(parallel).toMatchObject({ defaultValue: 4, min: 1, max: 32, step: 1 })
    expect(type?.rowFields).toStrictEqual([
      expect.objectContaining({ key: 'model', isRequired: true, optionsFrom: 'llm.models' }),
    ])
  })

  it('lists the titles the plugins give operations', async () => {
    const { operations } = await createConnectionClient(createRpc()).listTypes()

    expect(operations).toStrictEqual([{ name: 'llm.embed', title: 'embed text' }])
  })

  it('lists the connections with plain values', async () => {
    const connections = await createConnectionClient(createRpc()).list()

    expect(connections).toStrictEqual([
      {
        id: 'c00000001',
        typeId: 'md.numen.connections.llm/openrouter',
        name: 'Work',
        revision: 3,
        fields: { parallelRequests: 8, stream: true, label: 'a' },
        secretKeysSet: ['apiKey'],
        isTypeAvailable: true,
        isComplete: true,
        isBuiltIn: false,
        uses: [
          {
            pluginId: 'md.numen.tools.proofreading',
            pluginName: 'Proofreading',
            operations: ['llm.complete'],
          },
        ],
      },
    ])
  })

  it('sends the fields as values and the secrets as texts when it creates a connection', async () => {
    const requests: unknown[] = []
    const client = createConnectionClient(
      createRpc({
        createProviderConnection: (request) => {
          requests.push({
            typeId: request.typeId,
            name: request.name,
            fields: Object.fromEntries(
              Object.entries(request.fields ?? {}).map(([k, v]) => [
                k,
                toJson(ValueSchema, create(ValueSchema, v)),
              ]),
            ),
            secrets: request.secrets,
          })
          return Promise.resolve(
            create(CreateProviderConnectionResponseSchema, { connection: { id: 'c00000002' } }),
          )
        },
      }),
    )

    const answer = await client.create({
      typeId: 'a.b/x',
      name: 'N',
      fields: { baseUrl: 'https://a.example', parallelRequests: 2, stream: false },
      secrets: { apiKey: 'sk-1' },
    })

    expect(answer.ok).toBe(true)
    expect(requests).toStrictEqual([
      {
        typeId: 'a.b/x',
        name: 'N',
        fields: { baseUrl: 'https://a.example', parallelRequests: 2, stream: false },
        secrets: { apiKey: 'sk-1' },
      },
    ])
  })

  it('sends the revision it read when it updates a connection', async () => {
    const revisions: number[] = []
    const client = createConnectionClient(
      createRpc({
        updateProviderConnection: (request) => {
          revisions.push(request.revision ?? 0)
          return Promise.resolve(
            create(UpdateProviderConnectionResponseSchema, {
              connection: { id: request.id ?? '', revision: 4 },
            }),
          )
        },
      }),
    )

    const answer = await client.update({
      id: 'c00000001',
      revision: 3,
      name: 'N',
      fields: {},
      secrets: {},
    })

    expect(revisions).toStrictEqual([3])
    expect(answer.ok && answer.value.revision).toBe(4)
  })

  it('reads each served error code as a failure with its detail', async () => {
    const codes: Array<[ProviderConnectionErrorCode, string]> = [
      [ProviderConnectionErrorCode.NOT_FOUND, 'not_found'],
      [ProviderConnectionErrorCode.NAME_TAKEN, 'name_taken'],
      [ProviderConnectionErrorCode.INVALID_FIELD, 'invalid_field'],
      [ProviderConnectionErrorCode.TYPE_UNAVAILABLE, 'type_unavailable'],
      [ProviderConnectionErrorCode.KEY_STORE_FAILED, 'key_store_failed'],
      [ProviderConnectionErrorCode.STALE_REVISION, 'stale_revision'],
      [ProviderConnectionErrorCode.PROVIDER_UNAVAILABLE, 'provider_unavailable'],
    ]
    for (const [code, expected] of codes) {
      const client = createConnectionClient(
        createRpc({
          createProviderConnection: () =>
            Promise.resolve(
              create(CreateProviderConnectionResponseSchema, { error: code, detail: 'why' }),
            ),
        }),
      )

      const answer = await client.create({
        typeId: 'a.b/x',
        name: '',
        fields: {},
        secrets: {},
      })

      expect(answer).toStrictEqual({ ok: false, error: { errorCode: expected, detail: 'why' } })
    }
  })

  it('reads a failed delete as a failure', async () => {
    const client = createConnectionClient(
      createRpc({
        deleteProviderConnection: () =>
          Promise.resolve(
            create(DeleteProviderConnectionResponseSchema, {
              error: ProviderConnectionErrorCode.NOT_FOUND,
              detail: 'gone',
            }),
          ),
      }),
    )

    expect(await client.delete('c00000009')).toStrictEqual({
      ok: false,
      error: { errorCode: 'not_found', detail: 'gone' },
    })
    expect(await createConnectionClient(createRpc()).delete('c00000001')).toStrictEqual({
      ok: true,
      value: undefined,
    })
  })

  it('lists the options of a row field with the values of the row and the operations of the setting', async () => {
    const requests: unknown[] = []
    const client = createConnectionClient(
      createRpc({
        listRowOptions: (request) => {
          requests.push(request)
          return Promise.resolve(
            create(ListRowOptionsResponseSchema, {
              options: [
                { value: 'low', title: 'Low', group: '' },
                { value: 'qwen2.5:14b', title: '', group: 'Local' },
              ],
            }),
          )
        },
      }),
    )

    const answer = await client.listRowOptions({
      connectionId: 'c00000001',
      field: 'reasoning',
      options: { model: 'm' },
      operations: ['llm.complete'],
    })

    expect(requests).toStrictEqual([
      expect.objectContaining({
        connectionId: 'c00000001',
        field: 'reasoning',
        options: { model: 'm' },
        operations: ['llm.complete'],
      }),
    ])
    expect(answer).toStrictEqual({
      ok: true,
      value: [
        { value: 'low', title: 'Low', group: '' },
        { value: 'qwen2.5:14b', title: 'qwen2.5:14b', group: 'Local' },
      ],
    })
  })

  it('reads a provider that cannot list as a failure', async () => {
    const client = createConnectionClient(
      createRpc({
        listRowOptions: () =>
          Promise.resolve(
            create(ListRowOptionsResponseSchema, {
              error: ProviderConnectionErrorCode.PROVIDER_UNAVAILABLE,
              detail: 'key rejected',
            }),
          ),
      }),
    )

    expect(
      await client.listRowOptions({
        connectionId: 'c00000001',
        field: 'model',
        options: {},
        operations: [],
      }),
    ).toStrictEqual({
      ok: false,
      error: { errorCode: 'provider_unavailable', detail: 'key rejected' },
    })
  })
})
