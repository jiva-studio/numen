import { describe, expect, it } from 'vitest'
import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import type { ConnectionCallOptions, ConnectionError, ConnectionHost } from '../../connections'
import type { Result } from '../../result'
import {
  chat,
  ChatFinishSchema,
  ChatFrameSchema,
  ChatRequestSchema,
  complete,
  CompleteRequestSchema,
  CompleteResultSchema,
  embed,
  FinishReason,
  getProfile,
  ProfileSchema,
} from './index'

interface Asked {
  readonly operation: string
  readonly payload: Uint8Array
  readonly options: ConnectionCallOptions | undefined
}

const createHost = (
  frames: readonly Uint8Array[],
  end: Result<Uint8Array, ConnectionError>,
): { host: ConnectionHost; asked: Asked[] } => {
  const asked: Asked[] = []
  return {
    asked,
    host: {
      call: async function* (operation, payload, options) {
        asked.push({ operation, payload, options })
        yield* frames
        return end
      },
    },
  }
}

const failure: ConnectionError = {
  error: 'all_failed',
  message: 'Ollama: unavailable',
  failures: [{ connectionName: 'Ollama', reason: 'unavailable' }],
}

describe('the llm contract', () => {
  it('completes messages through llm.complete with the priority of the call', async () => {
    const result = toBinary(CompleteResultSchema, create(CompleteResultSchema, { text: 'fixed' }))
    const { host, asked } = createHost([], { ok: true, value: result })

    const answer = await complete(
      host,
      { messages: [{ role: 'user', text: 'helo' }], maxTokens: 30 },
      { priority: 'interactive' },
    )

    expect(answer.ok && answer.value.text).toBe('fixed')
    expect(asked[0]?.operation).toBe('llm.complete')
    expect(asked[0]?.options).toStrictEqual({ priority: 'interactive' })
    const sent = fromBinary(CompleteRequestSchema, asked[0]!.payload)
    expect(sent.messages.map((one) => one.text)).toStrictEqual(['helo'])
    expect(sent.maxTokens).toBe(30)
  })

  it('passes the error of the call on', async () => {
    const { host } = createHost([], { ok: false, error: failure })

    expect(await complete(host, { messages: [] })).toStrictEqual({ ok: false, error: failure })
  })

  it('makes no call for no text to embed', async () => {
    const { host, asked } = createHost([], { ok: false, error: failure })

    const answer = await embed(host, { texts: [] })

    expect(answer.ok && answer.value.vectors).toStrictEqual([])
    expect(asked).toStrictEqual([])
  })

  it('asks the profile in the background', async () => {
    const profile = toBinary(ProfileSchema, create(ProfileSchema, { width: 8, model: 'm' }))
    const { host, asked } = createHost([], { ok: true, value: profile })

    const answer = await getProfile(host)

    expect(answer.ok && answer.value.width).toBe(8)
    expect(asked[0]?.operation).toBe('llm.profile')
    expect(asked[0]?.options?.priority).toBe('background')
  })

  it('yields the frames of a turn and returns its finish', async () => {
    const delta = toBinary(
      ChatFrameSchema,
      create(ChatFrameSchema, { event: { case: 'textDelta', value: 'Hello' } }),
    )
    const finish = toBinary(
      ChatFinishSchema,
      create(ChatFinishSchema, { reason: FinishReason.TOOL_CALLS }),
    )
    const { host, asked } = createHost([delta], { ok: true, value: finish })

    const turn = chat(host, { messages: [{ role: 'user', text: 'hi' }] })
    const first = await turn.next()
    const last = await turn.next()

    expect(first.done === false && first.value.event).toStrictEqual({
      case: 'textDelta',
      value: 'Hello',
    })
    expect(last.done === true && last.value.ok && last.value.value.reason).toBe(
      FinishReason.TOOL_CALLS,
    )
    expect(asked[0]?.operation).toBe('llm.chat')
    expect(fromBinary(ChatRequestSchema, asked[0]!.payload).messages[0]?.text).toBe('hi')
  })
})
