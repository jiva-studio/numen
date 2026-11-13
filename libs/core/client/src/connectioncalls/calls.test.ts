import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CallErrorCode,
  CallFailureSchema,
  ConnectionCallServiceCallResponseSchema,
  FailureReason,
  ConnectionCallServiceListRowOptionsResponseSchema,
  ConnectionCallServiceListRowsResponseSchema,
  Priority,
  type ConnectionCallServiceCallResponse,
} from '@numen/protocol'
import { createConnectionCallClient, type ConnectionCallRpc } from './calls'

type Frame = ConnectionCallServiceCallResponse['frame']

const streamOf = (...frames: Frame[]) =>
  (async function* () {
    for (const frame of frames) yield create(ConnectionCallServiceCallResponseSchema, { frame })
  })()

const read = async (generator: AsyncGenerator<Uint8Array, unknown>) => {
  const data: Uint8Array[] = []
  for (;;) {
    const next = await generator.next()
    if (next.done === true) return { data, end: next.value }
    data.push(next.value)
  }
}

const bytes = (text: string) => new TextEncoder().encode(text)

const NO_LISTS: Omit<ConnectionCallRpc, 'call'> = {
  listRows: () => Promise.reject(new Error('no rows')),
  listRowOptions: () => Promise.reject(new Error('no options')),
}

const CHAT_REQUEST = {
  pluginId: 'md.numen.agents.numen',
  operation: 'llm.chat',
  payload: bytes(''),
}

describe('createConnectionCallClient', () => {
  it('sends the call of the plugin with its priority and signal, and yields the frames in order', async () => {
    const requests: unknown[] = []
    const options: unknown[] = []
    const rpc: ConnectionCallRpc = {
      ...NO_LISTS,
      call: (request, callOptions) => {
        requests.push(request)
        options.push(callOptions)
        return streamOf(
          { case: 'data', value: bytes('one') },
          { case: 'data', value: bytes('two') },
          { case: 'done', value: bytes('end') },
        )
      },
    }
    const controller = new AbortController()

    const { data, end } = await read(
      createConnectionCallClient(rpc).call(
        { pluginId: 'md.numen.tools.proofreading', operation: 'llm.chat', payload: bytes('ask') },
        {
          priority: 'interactive',
          signal: controller.signal,
        },
      ),
    )

    expect(data).toStrictEqual([bytes('one'), bytes('two')])
    expect(end).toStrictEqual({ ok: true, value: bytes('end') })
    expect(requests).toStrictEqual([
      expect.objectContaining({
        pluginId: 'md.numen.tools.proofreading',
        operation: 'llm.chat',
        payload: bytes('ask'),
        priority: Priority.INTERACTIVE,
      }),
    ])
    expect(options).toStrictEqual([expect.objectContaining({ signal: controller.signal })])
  })

  it('sends background where the call names no priority', async () => {
    const priorities: Priority[] = []
    const rpc: ConnectionCallRpc = {
      ...NO_LISTS,
      call: (request) => {
        priorities.push(request.priority ?? Priority.UNSPECIFIED)
        return streamOf({ case: 'done', value: new Uint8Array() })
      },
    }

    await read(
      createConnectionCallClient(rpc).call({
        pluginId: 'md.numen.engines.embeddings',
        operation: 'llm.embed',
        payload: bytes(''),
      }),
    )

    expect(priorities).toStrictEqual([Priority.BACKGROUND])
  })

  it('ends the call with the failure the core serves and keeps the frames before it', async () => {
    const codes: Array<[CallErrorCode, string]> = [
      [CallErrorCode.NO_CONNECTION, 'no_connection'],
      [CallErrorCode.ALL_FAILED, 'all_failed'],
      [CallErrorCode.STOPPED, 'stopped'],
    ]
    for (const [code, expected] of codes) {
      const rpc: ConnectionCallRpc = {
        ...NO_LISTS,
        call: () =>
          streamOf(
            { case: 'data', value: bytes('partial') },
            {
              case: 'failure',
              value: {
                $typeName: 'numen.v1.CallFailure',
                error: code,
                detail: 'Ollama: unavailable',
                failures: [
                  {
                    $typeName: 'numen.v1.CallRowFailure',
                    connectionName: 'Ollama',
                    reason: 'unavailable',
                    code: FailureReason.UNAVAILABLE,
                  },
                ],
              },
            },
          ),
      }

      const { data, end } = await read(createConnectionCallClient(rpc).call(CHAT_REQUEST))

      expect(data).toStrictEqual([bytes('partial')])
      expect(end).toStrictEqual({
        ok: false,
        error: {
          errorCode: expected,
          detail: 'Ollama: unavailable',
          failures: [{ connectionName: 'Ollama', reason: 'unavailable', code: 'unavailable' }],
        },
      })
    }
  })

  it('reads the reason of each failed row as its code, apart from its words', async () => {
    const codes: Array<[FailureReason, string]> = [
      [FailureReason.REJECTED_KEY, 'rejected_key'],
      [FailureReason.PAYMENT_REQUIRED, 'payment_required'],
      [FailureReason.RATE_LIMITED, 'rate_limited'],
      [FailureReason.INVALID_REQUEST, 'invalid_request'],
      [FailureReason.UNAVAILABLE, 'unavailable'],
    ]
    for (const [code, expected] of codes) {
      const failure = create(CallFailureSchema, {
        error: CallErrorCode.ALL_FAILED,
        failures: [{ connectionName: 'Harbor', reason: 'words of the reason', code }],
      })
      const rpc: ConnectionCallRpc = {
        ...NO_LISTS,
        call: () => streamOf({ case: 'failure', value: failure }),
      }

      const { end } = await read(createConnectionCallClient(rpc).call(CHAT_REQUEST))

      expect(end).toMatchObject({
        ok: false,
        error: { failures: [{ reason: 'words of the reason', code: expected }] },
      })
    }
  })

  it('throws where the stream ends before the call did', async () => {
    const rpc: ConnectionCallRpc = {
      ...NO_LISTS,
      call: () => streamOf({ case: 'data', value: bytes('cut') }),
    }

    await expect(read(createConnectionCallClient(rpc).call(CHAT_REQUEST))).rejects.toThrow(
      'before the call did',
    )
  })

  it('sends the row the call prefers with whether it is exclusive, and none where it prefers none', async () => {
    const requests: Array<{ preferred?: unknown }> = []
    const rpc: ConnectionCallRpc = {
      ...NO_LISTS,
      call: (request) => {
        requests.push(request)
        return streamOf({ case: 'done', value: new Uint8Array() })
      },
    }
    const client = createConnectionCallClient(rpc)

    await read(
      client.call(CHAT_REQUEST, {
        preferred: { connectionId: 'c-claude', options: { model: 'opus' } },
      }),
    )
    await read(client.call(CHAT_REQUEST))
    await read(
      client.call(CHAT_REQUEST, {
        preferred: { connectionId: 'c-claude', options: {}, isExclusive: true },
      }),
    )

    expect(requests[0]?.preferred).toStrictEqual({
      connectionId: 'c-claude',
      options: { model: 'opus' },
      isExclusive: false,
    })
    expect(requests[1]?.preferred).toBeUndefined()
    expect(requests[2]?.preferred).toStrictEqual({
      connectionId: 'c-claude',
      options: {},
      isExclusive: true,
    })
  })

  it('lists the rows of the plugin with their connection and row fields', async () => {
    const asked: unknown[] = []
    const rpc: ConnectionCallRpc = {
      ...NO_LISTS,
      call: () => streamOf(),
      listRows: (request) => {
        asked.push(request)
        return Promise.resolve(
          create(ConnectionCallServiceListRowsResponseSchema, {
            rows: [
              {
                connectionId: 'c-claude',
                connectionName: 'Claude',
                options: { model: 'opus' },
                fields: [
                  { key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: true },
                ],
              },
            ],
          }),
        )
      },
    }

    const rows = await createConnectionCallClient(rpc).listRows(
      'md.numen.views.agent.chat',
      'acp.task',
    )

    expect(asked).toStrictEqual([{ pluginId: 'md.numen.views.agent.chat', operation: 'acp.task' }])
    expect(rows).toStrictEqual([
      {
        connectionId: 'c-claude',
        connectionName: 'Claude',
        options: { model: 'opus' },
        fields: [{ key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: true }],
      },
    ])
  })

  it('lists the values of a row field', async () => {
    const asked: unknown[] = []
    const rpc: ConnectionCallRpc = {
      ...NO_LISTS,
      call: () => streamOf(),
      listRowOptions: (request) => {
        asked.push(request)
        return Promise.resolve(
          create(ConnectionCallServiceListRowOptionsResponseSchema, {
            options: [{ value: 'high', title: 'High', group: '' }],
          }),
        )
      },
    }
    const request = {
      pluginId: 'md.numen.views.agent.chat',
      operation: 'acp.task',
      connectionId: 'c-claude',
      field: 'reasoning',
      options: { model: 'opus' },
    }

    const options = await createConnectionCallClient(rpc).listRowOptions(request)

    expect(asked).toStrictEqual([request])
    expect(options).toStrictEqual([{ value: 'high', title: 'High', group: '' }])
  })
})
