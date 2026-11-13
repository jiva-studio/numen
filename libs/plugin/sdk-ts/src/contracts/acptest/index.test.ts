import { describe, expect, it } from 'vitest'
import { runTask, StopReason, type TaskFrame, type TaskRequest } from '../acp'
import { createTaskHost } from './index'

const collect = async (frames: AsyncGenerator<TaskFrame, unknown>) => {
  const texts: string[] = []
  for (;;) {
    const next = await frames.next()
    if (next.done === true) return { texts, end: next.value }
    if (next.value.update.case === 'agentMessageChunk') texts.push(next.value.update.value.text)
  }
}

describe('createTaskHost', () => {
  it('passes the request to the served task and returns its frames and stop reason to runTask', async () => {
    const asked: TaskRequest[] = []
    const host = createTaskHost(async function* (request) {
      asked.push(request)
      yield { update: { case: 'agentMessageChunk', value: { messageId: 'm', text: 'Two notes.' } } }
      return { ok: true, value: StopReason.MAX_TOKENS }
    })

    const { texts, end } = await collect(
      runTask(host, { prompt: 'what is here?', conversation: 'c1' }),
    )

    expect(asked.map((one) => [one.prompt, one.conversation])).toEqual([['what is here?', 'c1']])
    expect(texts).toEqual(['Two notes.'])
    expect(end).toMatchObject({ ok: true, value: { stopReason: StopReason.MAX_TOKENS } })
  })

  it('passes a thought and the diff of a call being written as they are served', async () => {
    const host = createTaskHost(async function* () {
      yield {
        update: {
          case: 'agentThoughtChunk',
          value: { messageId: 't1', text: 'The note has two beds.' },
        },
      }
      yield {
        update: {
          case: 'toolCallUpdate',
          value: { toolCallId: 'c1', diffs: [{ path: 'Garden/Beds.md', newText: '# Beds' }] },
        },
      }
      return { ok: true, value: StopReason.END_TURN }
    })

    const frames: TaskFrame[] = []
    const run = runTask(host, { prompt: 'plan the beds' })
    for (;;) {
      const next = await run.next()
      if (next.done === true) break
      frames.push(next.value)
    }

    expect(frames.map((frame) => frame.update)).toMatchObject([
      { case: 'agentThoughtChunk', value: { messageId: 't1', text: 'The note has two beds.' } },
      {
        case: 'toolCallUpdate',
        value: {
          toolCallId: 'c1',
          diffs: [{ path: 'Garden/Beds.md', oldText: '', newText: '# Beds' }],
        },
      },
    ])
  })

  it('returns the error the served task ends with', async () => {
    const host = createTaskHost(async function* () {
      yield* []
      return {
        ok: false,
        error: { error: 'all_failed', message: 'the agent is down', failures: [] },
      }
    })

    const { end } = await collect(runTask(host, { prompt: 'hi' }))

    expect(end).toEqual({
      ok: false,
      error: { error: 'all_failed', message: 'the agent is down', failures: [] },
    })
  })

  it('ends an operation other than acp.task with no_connection', async () => {
    const host = createTaskHost(async function* () {
      yield* []
      return { ok: true, value: StopReason.END_TURN }
    })

    const result = await host.call('llm.chat', new Uint8Array()).next()

    expect(result).toMatchObject({
      done: true,
      value: { ok: false, error: { error: 'no_connection' } },
    })
  })
})
