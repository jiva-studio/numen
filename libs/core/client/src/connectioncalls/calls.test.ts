import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CallErrorCode,
  ConnectionCallServiceCallResponseSchema,
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

const CHAT_REQUEST = { pluginId: 'md.numen.agents.numen', operation: 'llm.chat', payload: bytes('') }

describe('createConnectionCallClient', () => {
  it('sends the call of the plugin with its priority and signal, and yields the frames in order', async () => {
    const requests: unknown[] = []
    const options: unknown[] = []
    const rpc: ConnectionCallRpc = {
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
                  },
                ],
              },
            },
          ),
      }

      const { data, end } = await read(
        createConnectionCallClient(rpc).call(CHAT_REQUEST),
      )

      expect(data).toStrictEqual([bytes('partial')])
      expect(end).toStrictEqual({
        ok: false,
        error: {
          errorCode: expected,
          detail: 'Ollama: unavailable',
          failures: [{ connectionName: 'Ollama', reason: 'unavailable' }],
        },
      })
    }
  })

  it('throws where the stream ends before the call did', async () => {
    const rpc: ConnectionCallRpc = { call: () => streamOf({ case: 'data', value: bytes('cut') }) }

    await expect(
      read(createConnectionCallClient(rpc).call(CHAT_REQUEST)),
    ).rejects.toThrow('before the call did')
  })
})
