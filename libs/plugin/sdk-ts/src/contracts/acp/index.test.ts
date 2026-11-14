import { describe, expect, it } from 'vitest'
import { create, toBinary } from '@bufbuild/protobuf'
import type { ConnectionHost } from '../../connections'
import { runTask, StopReason, TaskDoneSchema, TaskFrameSchema } from './index'

const frame = toBinary(
  TaskFrameSchema,
  create(TaskFrameSchema, {
    update: { case: 'agentMessageChunk', value: { messageId: 'm', text: 'Two notes.' } },
  }),
)

describe('runTask', () => {
  it('returns the stop reason of a stream that ends and leaves the call to end itself', async () => {
    const done = toBinary(TaskDoneSchema, create(TaskDoneSchema, { stopReason: StopReason.END_TURN }))
    const host: ConnectionHost = {
      call: async function* () {
        yield frame
        return { ok: true, value: done }
      },
    }

    const run = runTask(host, { prompt: 'hi' })
    await run.next()
    const last = await run.next()

    expect(last).toMatchObject({
      done: true,
      value: { ok: true, value: { stopReason: StopReason.END_TURN } },
    })
  })

  it('throws on a frame that is no message of the contract and ends the call', async () => {
    let isEnded = false
    const host: ConnectionHost = {
      call: async function* () {
        try {
          yield new Uint8Array([0xff, 0xff, 0xff])
          return { ok: true, value: new Uint8Array() }
        } finally {
          isEnded = true
        }
      },
    }

    await expect(runTask(host, { prompt: 'hi' }).next()).rejects.toThrow()
    expect(isEnded).toBe(true)
  })

  it('ends the call when the reader stops before the stop reason', async () => {
    let isEnded = false
    const host: ConnectionHost = {
      call: async function* () {
        try {
          yield frame
          yield frame
          return { ok: true, value: new Uint8Array() }
        } finally {
          isEnded = true
        }
      },
    }

    const run = runTask(host, { prompt: 'hi' })
    await run.next()
    await run.return({ ok: true, value: create(TaskDoneSchema) })

    expect(isEnded).toBe(true)
  })
})
