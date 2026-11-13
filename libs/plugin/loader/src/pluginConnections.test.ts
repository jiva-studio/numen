import { describe, expect, it, vi } from 'vitest'
import type { ConnectionCallClient } from '@numen/core-client'
import { createPluginConnections } from './pluginConnections'

const bytes = (text: string) => new TextEncoder().encode(text)

const NO_LISTS: Omit<ConnectionCallClient, 'call'> = {
  listRows: () => Promise.reject(new Error('no rows')),
  listRowOptions: () => Promise.reject(new Error('no options')),
}

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
      ...NO_LISTS,
      call: async function* (request, options) {
        requests.push({ ...request, options })
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
      ...NO_LISTS,
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
      ...NO_LISTS,
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
      ...NO_LISTS,
      call: async function* () {
        yield* []
        throw new Error('the core is gone')
      },
    }

    await expect(
      read(createPluginConnections(client, 'md.numen.a').call('llm.chat', bytes(''))),
    ).rejects.toThrow('the core is gone')
  })

  it('lists the rows and the values of a row field of the plugin', async () => {
    const asked: unknown[] = []
    const row = {
      connectionId: 'c-claude',
      connectionName: 'Claude',
      options: { model: 'opus' },
      fields: [{ key: 'model', title: 'Model', optionsFrom: 'llm.models' }],
    }
    const client: ConnectionCallClient = {
      call: async function* () {
        yield* []
        return { ok: true, value: bytes('') }
      },
      listRows: (pluginId, operation) => {
        asked.push({ pluginId, operation })
        return Promise.resolve([row])
      },
      listRowOptions: (request) => {
        asked.push(request)
        return Promise.resolve([{ value: 'high', title: 'High', group: '' }])
      },
    }
    const connections = createPluginConnections(client, 'md.numen.views.agent.chat')

    const rows = await connections.listRows?.('acp.task')
    const options = await connections.listRowOptions?.({
      operation: 'acp.task',
      connectionId: 'c-claude',
      field: 'reasoning',
      options: { model: 'opus' },
    })

    expect(rows).toStrictEqual([row])
    expect(options).toStrictEqual([{ value: 'high', title: 'High', group: '' }])
    expect(asked).toStrictEqual([
      { pluginId: 'md.numen.views.agent.chat', operation: 'acp.task' },
      {
        pluginId: 'md.numen.views.agent.chat',
        operation: 'acp.task',
        connectionId: 'c-claude',
        field: 'reasoning',
        options: { model: 'opus' },
      },
    ])
  })

  it('tells the plugin of each change the window signals until the plugin stops listening', () => {
    const handlers = new Set<() => void>()
    const subscribe = (handler: () => void) => {
      handlers.add(handler)
      return () => handlers.delete(handler)
    }
    const connections = createPluginConnections(
      { ...NO_LISTS, call: vi.fn() },
      'md.numen.views.agent.chat',
      subscribe,
    )
    const changed = vi.fn()

    const listening = connections.onChange?.(changed)
    for (const handler of handlers) handler()
    listening?.dispose()
    for (const handler of handlers) handler()

    expect(changed).toHaveBeenCalledTimes(1)
  })

  it('offers no change signal where the window gives none', () => {
    const connections = createPluginConnections(
      { ...NO_LISTS, call: vi.fn() },
      'md.numen.views.agent.chat',
    )

    expect(connections.onChange).toBeUndefined()
  })
})
