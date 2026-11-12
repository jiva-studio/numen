import { describe, expect, it } from 'vitest'
import type { ConnectionCallClient } from '@numen/core-client'
import { createPluginConnections } from './pluginConnections'

const bytes = (text: string) => new TextEncoder().encode(text)

const read = async (generator: AsyncGenerator<Uint8Array, unknown>) => {
  const data: Uint8Array[] = []
  for (;;) {
    const next = await generator.next()
    if (next.done === true) return { data, end: next.value }
    data.push(next.value)
  }
}

describe('the connections of a plugin', () => {
  it('carries out an operation for the plugin and yields its frames', async () => {
    const requests: unknown[] = []
    const client: ConnectionCallClient = {
      call: async function* (pluginId, operation, payload, options) {
        requests.push({ pluginId, operation, payload, options })
        yield bytes('one')
        return { ok: true, value: bytes('end') }
      },
    }
    const controller = new AbortController()

    const { data, end } = await read(
      createPluginConnections(client, 'md.numen.tools.proofreading').call(
        'llm.complete',
        bytes('ask'),
        {
          priority: 'interactive',
          signal: controller.signal,
        },
      ),
    )

    expect(data).toStrictEqual([bytes('one')])
    expect(end).toStrictEqual({ ok: true, value: bytes('end') })
    expect(requests).toStrictEqual([
      {
        pluginId: 'md.numen.tools.proofreading',
        operation: 'llm.complete',
        payload: bytes('ask'),
        options: { priority: 'interactive', signal: controller.signal },
      },
    ])
  })

  it('reads the failure the core serves as the error of the call', async () => {
    const client: ConnectionCallClient = {
      call: async function* () {
        yield* []
        return {
          ok: false,
          error: {
            errorCode: 'all_failed',
            detail: 'Ollama: unavailable',
            failures: [{ connectionName: 'Ollama', reason: 'unavailable' }],
          },
        }
      },
    }

    const { end } = await read(
      createPluginConnections(client, 'md.numen.a').call('llm.chat', bytes('')),
    )

    expect(end).toStrictEqual({
      ok: false,
      error: {
        error: 'all_failed',
        message: 'Ollama: unavailable',
        failures: [{ connectionName: 'Ollama', reason: 'unavailable' }],
      },
    })
  })

  it('reads a call whose signal aborted as canceled', async () => {
    const controller = new AbortController()
    const client: ConnectionCallClient = {
      call: async function* () {
        yield* []
        controller.abort()
        throw new Error('aborted')
      },
    }

    const { end } = await read(
      createPluginConnections(client, 'md.numen.a').call('llm.chat', bytes(''), {
        signal: controller.signal,
      }),
    )

    expect(end).toStrictEqual({
      ok: false,
      error: { error: 'canceled', message: 'aborted', failures: [] },
    })
  })

  it('lets a call that fails for another reason throw', async () => {
    const client: ConnectionCallClient = {
      call: async function* () {
        yield* []
        throw new Error('the core is gone')
      },
    }

    await expect(
      read(createPluginConnections(client, 'md.numen.a').call('llm.chat', bytes(''))),
    ).rejects.toThrow('the core is gone')
  })
})
